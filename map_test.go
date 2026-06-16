package hamt

import (
	"fmt"
	"math/bits"
	"testing"
	"testing/quick"
)

func TestMapSetGetLen(t *testing.T) {
	m := NewMap[int, string](IntHasher{})

	if m.Len() != 0 {
		t.Fatalf("empty len = %d, want 0", m.Len())
	}
	if _, ok := m.Get(1); ok {
		t.Fatal("empty map returned key")
	}

	m = m.Set(1, "one")
	m = m.Set(2, "two")

	assertLen(t, m, 2)
	assertGet(t, m, 1, "one")
	assertGet(t, m, 2, "two")
	assertMissing[int, string](t, m, 3)
	validateMap(t, m)
}

func TestMapOverwriteDoesNotGrow(t *testing.T) {
	m := NewMap[int, string](IntHasher{}).Set(1, "one")
	n := m.Set(1, "uno")

	assertLen(t, n, 1)
	assertGet(t, n, 1, "uno")
	assertGet(t, m, 1, "one")
	validateMap(t, n)
}

func TestMapDelete(t *testing.T) {
	m := NewMap[int, string](IntHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")
	m = m.Set(3, "three")

	n := m.Delete(2)

	assertLen(t, n, 2)
	assertGet(t, n, 1, "one")
	assertMissing[int, string](t, n, 2)
	assertGet(t, n, 3, "three")
	assertGet(t, m, 2, "two")
	validateMap(t, n)
}

func TestMapDeleteMissingReturnsEquivalentMap(t *testing.T) {
	m := NewMap[int, string](IntHasher{}).Set(1, "one")
	n := m.Delete(2)

	if n.root != m.root {
		t.Fatal("delete of missing key should preserve root")
	}
	assertLen(t, n, 1)
	assertGet(t, n, 1, "one")
}

func TestMapSnapshotsAreImmutable(t *testing.T) {
	m0 := NewMap[int, string](IntHasher{})
	m1 := m0.Set(1, "one")
	m2 := m1.Set(1, "uno")
	m3 := m2.Delete(1)

	assertMissing[int, string](t, m0, 1)
	assertGet(t, m1, 1, "one")
	assertGet(t, m2, 1, "uno")
	assertMissing[int, string](t, m3, 1)

	validateMap(t, m1)
	validateMap(t, m2)
	validateMap(t, m3)
}

func TestMapForcedHashCollisions(t *testing.T) {
	m := NewMap[int, string](constantIntHasher{})
	for i := 0; i < 20; i++ {
		m = m.Set(i, fmt.Sprintf("value-%d", i))
	}

	assertLen(t, m, 20)
	for i := 0; i < 20; i++ {
		assertGet(t, m, i, fmt.Sprintf("value-%d", i))
	}

	m = m.Set(7, "updated")
	m = m.Delete(3)

	assertLen(t, m, 19)
	assertGet(t, m, 7, "updated")
	assertMissing[int, string](t, m, 3)
	validateMap(t, m)
}

func TestMapDeleteCanonicalizesSingletonCollision(t *testing.T) {
	m := NewMap[int, string](constantIntHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")

	m = m.Delete(2)

	assertLen(t, m, 1)
	assertGet(t, m, 1, "one")
	if m.root == nil || m.root.collision {
		t.Fatal("delete should collapse a two-entry collision node to a direct entry")
	}
	if _, ok := m.root.singleton(); !ok {
		t.Fatal("delete should leave a singleton root")
	}
	validateMap(t, m)
}

func TestMapRange(t *testing.T) {
	m := NewMap[int, string](IntHasher{})
	want := map[int]string{}
	for i := 0; i < 64; i++ {
		m = m.Set(i, fmt.Sprintf("value-%d", i))
		want[i] = fmt.Sprintf("value-%d", i)
	}

	got := map[int]string{}
	m.Range(func(k int, v string) bool {
		got[k] = v
		return true
	})

	if len(got) != len(want) {
		t.Fatalf("range count = %d, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("range[%d] = %q, want %q", k, got[k], v)
		}
	}
}

func TestMapRangeStops(t *testing.T) {
	m := NewMap[int, int](IntHasher{})
	for i := 0; i < 10; i++ {
		m = m.Set(i, i)
	}

	count := 0
	m.Range(func(int, int) bool {
		count++
		return false
	})
	if count != 1 {
		t.Fatalf("range count = %d, want 1", count)
	}
}

func TestMapQuickAgainstBuiltin(t *testing.T) {
	prop := func(ops []uint64) bool {
		m := NewMap[int, int](IntHasher{})
		want := map[int]int{}

		for _, op := range ops {
			key := int((op >> 2) % 97)
			value := int(op >> 9)
			switch op & 3 {
			case 0, 1:
				m = m.Set(key, value)
				want[key] = value
			case 2:
				m = m.Delete(key)
				delete(want, key)
			case 3:
				got, ok := m.Get(key)
				wantValue, wantOK := want[key]
				if ok != wantOK || got != wantValue {
					return false
				}
			}
			if !mapsEqual(m, want) {
				return false
			}
			if err := checkMap(m); err != nil {
				return false
			}
		}
		return true
	}

	if err := quick.Check(prop, &quick.Config{MaxCount: 200}); err != nil {
		t.Fatal(err)
	}
}

func TestNilHasherPanicsOnSet(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Set with nil hasher did not panic")
		}
	}()
	_ = NewMap[int, int](nil).Set(1, 1)
}

func assertLen[K, V any](t *testing.T, m Map[K, V], want int) {
	t.Helper()
	if m.Len() != want {
		t.Fatalf("Len() = %d, want %d", m.Len(), want)
	}
}

func assertGet[K comparable, V comparable](t *testing.T, m Map[K, V], key K, want V) {
	t.Helper()
	got, ok := m.Get(key)
	if !ok || got != want {
		t.Fatalf("Get(%v) = %v, %v; want %v, true", key, got, ok, want)
	}
}

func assertMissing[K comparable, V any](t *testing.T, m Map[K, V], key K) {
	t.Helper()
	if got, ok := m.Get(key); ok {
		t.Fatalf("Get(%v) = %v, true; want missing", key, got)
	}
}

func validateMap[K comparable, V comparable](t *testing.T, m Map[K, V]) {
	t.Helper()
	if err := checkMap(m); err != nil {
		t.Fatal(err)
	}
}

func mapsEqual[K comparable, V comparable](m Map[K, V], want map[K]V) bool {
	if m.Len() != len(want) {
		return false
	}
	seen := map[K]V{}
	m.Range(func(k K, v V) bool {
		seen[k] = v
		return true
	})
	if len(seen) != len(want) {
		return false
	}
	for k, wantValue := range want {
		got, ok := seen[k]
		if !ok || got != wantValue {
			return false
		}
	}
	return true
}

func checkMap[K comparable, V comparable](m Map[K, V]) error {
	if m.root == nil {
		if m.Len() != 0 {
			return fmt.Errorf("nil root len = %d", m.Len())
		}
		return nil
	}
	count, err := checkNode(m.root, 0, m.hasher)
	if err != nil {
		return err
	}
	if count != m.Len() {
		return fmt.Errorf("reachable count = %d, len = %d", count, m.Len())
	}
	return nil
}

func checkNode[K comparable, V comparable](n *node[K, V], shift uint, h Hasher[K]) (int, error) {
	if n == nil {
		return 0, fmt.Errorf("nil child")
	}
	if n.collision {
		if len(n.collisions) < 2 {
			return 0, fmt.Errorf("collision len = %d, want >= 2", len(n.collisions))
		}
		seen := map[K]struct{}{}
		for _, e := range n.collisions {
			if e.hash != n.collisionHash {
				return 0, fmt.Errorf("collision hash = %d, want %d", e.hash, n.collisionHash)
			}
			if h.Hash(e.key) != e.hash {
				return 0, fmt.Errorf("stored hash mismatch")
			}
			if _, ok := seen[e.key]; ok {
				return 0, fmt.Errorf("duplicate collision key")
			}
			seen[e.key] = struct{}{}
		}
		return len(n.collisions), nil
	}

	if n.dataMap&n.nodeMap != 0 {
		return 0, fmt.Errorf("data and node bitmaps overlap")
	}
	if bits.OnesCount32(n.dataMap) != len(n.entries) {
		return 0, fmt.Errorf("data bitmap count = %d, entries = %d", bits.OnesCount32(n.dataMap), len(n.entries))
	}
	if bits.OnesCount32(n.nodeMap) != len(n.children) {
		return 0, fmt.Errorf("node bitmap count = %d, children = %d", bits.OnesCount32(n.nodeMap), len(n.children))
	}
	if n.dataMap == 0 && n.nodeMap == 0 {
		return 0, fmt.Errorf("empty branch")
	}

	count := 0
	entryIdx := 0
	for bit := uint32(1); bit != 0; bit <<= 1 {
		if n.dataMap&bit == 0 {
			continue
		}
		e := n.entries[entryIdx]
		entryIdx++
		if bitpos(fragment(e.hash, shift)) != bit {
			return 0, fmt.Errorf("entry stored under wrong fragment")
		}
		if h.Hash(e.key) != e.hash {
			return 0, fmt.Errorf("stored hash mismatch")
		}
		count++
	}

	childIdx := 0
	for bit := uint32(1); bit != 0; bit <<= 1 {
		if n.nodeMap&bit == 0 {
			continue
		}
		child := n.children[childIdx]
		childIdx++
		childCount, err := checkNode(child, shift+fragmentBits, h)
		if err != nil {
			return 0, err
		}
		if childCount == 1 {
			return 0, fmt.Errorf("uncanonical singleton child")
		}
		count += childCount
	}

	return count, nil
}

type constantIntHasher struct{}

func (constantIntHasher) Hash(int) uint64     { return 1 }
func (constantIntHasher) Equal(a, b int) bool { return a == b }
