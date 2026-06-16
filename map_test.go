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

func TestMapSetInsertDoesNotMutateSnapshot(t *testing.T) {
	m := NewMap[uint64, string](identityUint64Hasher{})
	m = m.Set(1, "one")

	n := m.Set(2, "two")

	assertLen(t, m, 1)
	assertGet(t, m, 1, "one")
	assertMissing[uint64, string](t, m, 2)
	assertLen(t, n, 2)
	assertGet(t, n, 1, "one")
	assertGet(t, n, 2, "two")
	validateMap(t, m)
	validateMap(t, n)
}

func TestMapSetEntryToBranchDoesNotMutateSnapshot(t *testing.T) {
	m := NewMap[uint64, string](identityUint64Hasher{})
	m = m.Set(0, "zero")

	n := m.Set(32, "thirty-two")

	assertLen(t, m, 1)
	assertGet(t, m, 0, "zero")
	assertMissing[uint64, string](t, m, 32)
	assertLen(t, n, 2)
	assertGet(t, n, 0, "zero")
	assertGet(t, n, 32, "thirty-two")
	validateMap(t, m)
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

	assertLen(t, n, 1)
	assertGet(t, n, 1, "one")
	validateMap(t, n)
}

func TestMapDeleteMissingSameFragmentReturnsEquivalentMap(t *testing.T) {
	m := NewMap[uint64, string](identityUint64Hasher{}).Set(0, "zero")
	n := m.Delete(32)

	assertLen(t, n, 1)
	assertGet(t, n, 0, "zero")
	validateMap(t, n)
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

func TestMapDeleteFromBranchDoesNotMutateSnapshot(t *testing.T) {
	m := NewMap[uint64, string](identityUint64Hasher{})
	m = m.Set(0, "zero")
	m = m.Set(32, "thirty-two")
	m = m.Set(64, "sixty-four")

	n := m.Delete(64)

	assertGet(t, m, 0, "zero")
	assertGet(t, m, 32, "thirty-two")
	assertGet(t, m, 64, "sixty-four")
	assertGet(t, n, 0, "zero")
	assertGet(t, n, 32, "thirty-two")
	assertMissing[uint64, string](t, n, 64)
	validateMap(t, m)
	validateMap(t, n)
}

func TestMapSetInBranchDoesNotMutateSnapshot(t *testing.T) {
	m := NewMap[uint64, string](identityUint64Hasher{})
	m = m.Set(0, "zero")
	m = m.Set(32, "thirty-two")
	m = m.Set(64, "sixty-four")

	n := m.Set(64, "updated")

	assertGet(t, m, 0, "zero")
	assertGet(t, m, 32, "thirty-two")
	assertGet(t, m, 64, "sixty-four")
	assertGet(t, n, 0, "zero")
	assertGet(t, n, 32, "thirty-two")
	assertGet(t, n, 64, "updated")
	validateMap(t, m)
	validateMap(t, n)
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

func TestMapCollisionGetHashMismatch(t *testing.T) {
	m := NewMap[int, string](splitCollisionHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")

	assertMissing[int, string](t, m, 3)
	validateMap(t, m)
}

func TestMapCollisionDeleteMissingReturnsEquivalentMap(t *testing.T) {
	m := NewMap[int, string](constantIntHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")
	n := m.Delete(3)

	assertLen(t, n, 2)
	assertGet(t, n, 1, "one")
	assertGet(t, n, 2, "two")
	validateMap(t, n)
}

func TestMapCollisionDeleteHashMismatchReturnsEquivalentMap(t *testing.T) {
	m := NewMap[int, string](splitCollisionHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")
	n := m.Delete(3)

	assertLen(t, n, 2)
	assertGet(t, n, 1, "one")
	assertGet(t, n, 2, "two")
	validateMap(t, n)
}

func TestMapCollisionInsertDoesNotMutateSnapshot(t *testing.T) {
	m := NewMap[int, string](constantIntHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")

	n := m.Set(3, "three")

	assertLen(t, m, 2)
	assertGet(t, m, 1, "one")
	assertGet(t, m, 2, "two")
	assertMissing[int, string](t, m, 3)
	assertLen(t, n, 3)
	assertGet(t, n, 1, "one")
	assertGet(t, n, 2, "two")
	assertGet(t, n, 3, "three")
	validateMap(t, m)
	validateMap(t, n)
}

func TestMapCollisionInsertFromBuilderDoesNotAliasSiblings(t *testing.T) {
	b := NewBuilder[int, string](constantIntHasher{})
	b.Set(1, "one")
	b.Set(2, "two")
	b.Set(3, "three")
	m := b.Map()

	n := m.Set(4, "four")
	p := m.Set(5, "five")

	assertLen(t, m, 3)
	assertMissing[int, string](t, m, 4)
	assertMissing[int, string](t, m, 5)
	assertLen(t, n, 4)
	assertGet(t, n, 4, "four")
	assertMissing[int, string](t, n, 5)
	assertLen(t, p, 4)
	assertMissing[int, string](t, p, 4)
	assertGet(t, p, 5, "five")
	validateMap(t, m)
	validateMap(t, n)
	validateMap(t, p)
}

func TestMapCollisionOverwriteDoesNotMutateSnapshot(t *testing.T) {
	m := NewMap[int, string](constantIntHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")
	m = m.Set(3, "three")

	n := m.Set(2, "updated")

	assertLen(t, m, 3)
	assertGet(t, m, 2, "two")
	assertLen(t, n, 3)
	assertGet(t, n, 2, "updated")
	validateMap(t, m)
	validateMap(t, n)
}

func TestMapCollisionDeleteDoesNotMutateSnapshot(t *testing.T) {
	m := NewMap[int, string](constantIntHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")
	m = m.Set(3, "three")

	n := m.Delete(2)

	assertLen(t, m, 3)
	assertGet(t, m, 2, "two")
	assertLen(t, n, 2)
	assertMissing[int, string](t, n, 2)
	assertGet(t, n, 1, "one")
	assertGet(t, n, 3, "three")
	validateMap(t, m)
	validateMap(t, n)
}

func TestMapCollisionExpansionPreservesExistingHashes(t *testing.T) {
	m := NewMap[int, string](splitCollisionHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")
	m = m.Set(3, "three")

	assertLen(t, m, 3)
	assertGet(t, m, 1, "one")
	assertGet(t, m, 2, "two")
	assertGet(t, m, 3, "three")
	validateMap(t, m)
}

func TestMapDeleteCanonicalizesSingletonCollision(t *testing.T) {
	m := NewMap[int, string](constantIntHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")

	m = m.Delete(2)

	assertLen(t, m, 1)
	assertGet(t, m, 1, "one")
	if m.root == nil || m.root.isCollision() {
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

func TestMapRangeForcedHashCollisions(t *testing.T) {
	m := NewMap[int, string](constantIntHasher{})
	want := map[int]string{}
	for i := 0; i < 8; i++ {
		value := fmt.Sprintf("value-%d", i)
		m = m.Set(i, value)
		want[i] = value
	}

	got := map[int]string{}
	visits := 0
	m.Range(func(k int, v string) bool {
		visits++
		if _, ok := got[k]; ok {
			t.Fatalf("Range visited key %d more than once", k)
		}
		got[k] = v
		return true
	})

	if visits != len(want) {
		t.Fatalf("range visits = %d, want %d", visits, len(want))
	}
	if len(got) != len(want) {
		t.Fatalf("range count = %d, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("range[%d] = %q, want %q", k, got[k], v)
		}
	}
	validateMap(t, m)
}

func TestMapRangeStopsInCollision(t *testing.T) {
	m := NewMap[int, int](constantIntHasher{})
	for i := 0; i < 4; i++ {
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
	validateMap(t, m)
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
	assertPanics(t, func() {
		_ = NewMap[int, int](nil).Set(1, 1)
	})
}

func TestNilBuilderHasherPanicsOnSet(t *testing.T) {
	b := NewBuilder[int, int](nil)
	assertPanics(t, func() { b.Set(1, 1) })
}

func TestBuilderSetGetLenMap(t *testing.T) {
	b := NewBuilder[int, string](IntHasher{})
	if b.Len() != 0 {
		t.Fatalf("empty builder len = %d, want 0", b.Len())
	}

	b.Set(1, "one")
	b.Set(2, "two")
	b.Set(1, "uno")

	if b.Len() != 2 {
		t.Fatalf("builder len = %d, want 2", b.Len())
	}
	assertBuilderGet(t, b, 1, "uno")
	assertBuilderGet(t, b, 2, "two")
	assertBuilderMissing[int, string](t, b, 3)

	m := b.Map()
	assertLen(t, m, 2)
	assertGet(t, m, 1, "uno")
	assertGet(t, m, 2, "two")
	validateMap(t, m)
}

func TestBuilderDelete(t *testing.T) {
	b := NewBuilder[int, string](IntHasher{})
	b.Set(1, "one")
	b.Set(2, "two")
	b.Set(3, "three")
	b.Delete(2)
	b.Delete(4)

	if b.Len() != 2 {
		t.Fatalf("builder len = %d, want 2", b.Len())
	}
	assertBuilderGet(t, b, 1, "one")
	assertBuilderMissing[int, string](t, b, 2)
	assertBuilderGet(t, b, 3, "three")

	m := b.Map()
	assertLen(t, m, 2)
	assertMissing[int, string](t, m, 2)
	validateMap(t, m)
}

func TestBuilderDeleteMissingSameFragment(t *testing.T) {
	b := NewBuilder[uint64, string](identityUint64Hasher{})
	b.Set(0, "zero")
	b.Delete(32)
	m := b.Map()

	assertLen(t, m, 1)
	assertGet(t, m, 0, "zero")
	validateMap(t, m)
}

func TestBuilderForcedHashCollisions(t *testing.T) {
	b := NewBuilder[int, string](constantIntHasher{})
	for i := 0; i < 20; i++ {
		b.Set(i, fmt.Sprintf("value-%d", i))
	}
	b.Set(7, "updated")
	b.Delete(3)

	m := b.Map()
	assertLen(t, m, 19)
	for i := 0; i < 20; i++ {
		if i == 3 {
			assertMissing[int, string](t, m, i)
			continue
		}
		want := fmt.Sprintf("value-%d", i)
		if i == 7 {
			want = "updated"
		}
		assertGet(t, m, i, want)
	}
	validateMap(t, m)
}

func TestBuilderDeleteMissingCollision(t *testing.T) {
	b := NewBuilder[int, string](constantIntHasher{})
	b.Set(1, "one")
	b.Set(2, "two")
	b.Delete(3)
	m := b.Map()

	assertLen(t, m, 2)
	assertGet(t, m, 1, "one")
	assertGet(t, m, 2, "two")
	validateMap(t, m)
}

func TestBuilderDeleteHashMismatchCollision(t *testing.T) {
	b := NewBuilder[int, string](splitCollisionHasher{})
	b.Set(1, "one")
	b.Set(2, "two")
	b.Delete(3)
	m := b.Map()

	assertLen(t, m, 2)
	assertGet(t, m, 1, "one")
	assertGet(t, m, 2, "two")
	validateMap(t, m)
}

func TestBuilderCollisionExpansionPreservesExistingHashes(t *testing.T) {
	b := NewBuilder[int, string](splitCollisionHasher{})
	b.Set(1, "one")
	b.Set(2, "two")
	b.Set(3, "three")
	m := b.Map()

	assertLen(t, m, 3)
	assertGet(t, m, 1, "one")
	assertGet(t, m, 2, "two")
	assertGet(t, m, 3, "three")
	validateMap(t, m)
}

func TestBuilderDeleteCanonicalizesSingletonCollision(t *testing.T) {
	b := NewBuilder[int, string](constantIntHasher{})
	b.Set(1, "one")
	b.Set(2, "two")
	b.Delete(2)

	m := b.Map()
	assertLen(t, m, 1)
	assertGet(t, m, 1, "one")
	if m.root == nil || m.root.isCollision() {
		t.Fatal("builder delete should collapse a two-entry collision node to a direct entry")
	}
	if _, ok := m.root.singleton(); !ok {
		t.Fatal("builder delete should leave a singleton root")
	}
	validateMap(t, m)
}

func TestBuilderMapInvalidatesBuilder(t *testing.T) {
	b := NewBuilder[int, string](IntHasher{})
	b.Set(1, "one")
	m := b.Map()
	assertGet(t, m, 1, "one")

	assertPanics(t, func() { b.Len() })
	assertPanics(t, func() { b.Get(1) })
	assertPanics(t, func() { b.Set(2, "two") })
	assertPanics(t, func() { b.Delete(1) })
	assertPanics(t, func() { b.Map() })
}

func TestBuilderMapResultIsImmutable(t *testing.T) {
	b := NewBuilder[int, string](IntHasher{})
	b.Set(1, "one")
	m := b.Map()
	n := m.Set(1, "uno")

	assertGet(t, m, 1, "one")
	assertGet(t, n, 1, "uno")
	validateMap(t, m)
	validateMap(t, n)
}

func TestBuilderCopyInvalidatedByMap(t *testing.T) {
	b := NewBuilder[int, string](IntHasher{})
	b.Set(1, "one")
	copied := *b

	m := b.Map()

	assertPanics(t, func() { copied.Delete(1) })
	assertGet(t, m, 1, "one")
	validateMap(t, m)
}

func TestBuilderQuickAgainstBuiltin(t *testing.T) {
	prop := func(ops []uint64) bool {
		b := NewBuilder[int, int](IntHasher{})
		want := map[int]int{}

		for _, op := range ops {
			key := int((op >> 2) % 97)
			value := int(op >> 9)
			switch op & 3 {
			case 0, 1:
				b.Set(key, value)
				want[key] = value
			case 2:
				b.Delete(key)
				delete(want, key)
			case 3:
				got, ok := b.Get(key)
				wantValue, wantOK := want[key]
				if ok != wantOK || got != wantValue {
					return false
				}
			}
			if b.Len() != len(want) {
				return false
			}
		}

		m := b.Map()
		return mapsEqual(m, want) && checkMap(m) == nil
	}

	if err := quick.Check(prop, &quick.Config{MaxCount: 200}); err != nil {
		t.Fatal(err)
	}
}

func TestUint64Hasher(t *testing.T) {
	h := Uint64Hasher{}
	if h.Hash(1) != h.Hash(1) {
		t.Fatal("Hash(1) is not stable")
	}
	if !h.Equal(1, 1) {
		t.Fatal("Equal(1, 1) = false, want true")
	}
	if h.Equal(1, 2) {
		t.Fatal("Equal(1, 2) = true, want false")
	}
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

func assertBuilderGet[K comparable, V comparable](t *testing.T, b *Builder[K, V], key K, want V) {
	t.Helper()
	got, ok := b.Get(key)
	if !ok || got != want {
		t.Fatalf("Builder.Get(%v) = %v, %v; want %v, true", key, got, ok, want)
	}
}

func assertBuilderMissing[K comparable, V any](t *testing.T, b *Builder[K, V], key K) {
	t.Helper()
	if got, ok := b.Get(key); ok {
		t.Fatalf("Builder.Get(%v) = %v, true; want missing", key, got)
	}
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
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
	if n.isCollision() {
		if len(n.collisions) < 2 {
			return 0, fmt.Errorf("collision len = %d, want >= 2", len(n.collisions))
		}
		seen := map[K]struct{}{}
		for _, e := range n.collisions {
			if h.Hash(e.key) != n.collisionHash {
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

type splitCollisionHasher struct{}

func (splitCollisionHasher) Hash(v int) uint64 {
	switch v {
	case 1, 2:
		return 0
	case 3:
		return 1 << fragmentBits
	default:
		return uint64(v) << fragmentBits
	}
}

func (splitCollisionHasher) Equal(a, b int) bool { return a == b }

type identityUint64Hasher struct{}

func (identityUint64Hasher) Hash(v uint64) uint64   { return v }
func (identityUint64Hasher) Equal(a, b uint64) bool { return a == b }
