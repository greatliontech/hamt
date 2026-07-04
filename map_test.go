package hamt

import (
	"fmt"
	"maps"
	"math"
	"math/bits"
	"sync"
	"testing"
	"testing/quick"
)

func TestMapSetGetLen(t *testing.T) {
	m := NewWithHasher[int, string](testIntHasher{})

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
	m := NewWithHasher[int, string](testIntHasher{}).Set(1, "one")
	n := m.Set(1, "uno")

	assertLen(t, n, 1)
	assertGet(t, n, 1, "uno")
	assertGet(t, m, 1, "one")
	validateMap(t, n)
}

func TestMapSetInsertDoesNotMutateSnapshot(t *testing.T) {
	m := NewWithHasher[uint64, string](identityUint64Hasher{})
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
	m := NewWithHasher[uint64, string](identityUint64Hasher{})
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
	m := NewWithHasher[int, string](testIntHasher{})
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
	m := NewWithHasher[int, string](testIntHasher{}).Set(1, "one")
	n := m.Delete(2)

	assertLen(t, n, 1)
	assertGet(t, n, 1, "one")
	validateMap(t, n)
}

func TestMapDeleteMissingSameFragmentReturnsEquivalentMap(t *testing.T) {
	m := NewWithHasher[uint64, string](identityUint64Hasher{}).Set(0, "zero")
	n := m.Delete(32)

	assertLen(t, n, 1)
	assertGet(t, n, 0, "zero")
	validateMap(t, n)
}

func TestMapSnapshotsAreImmutable(t *testing.T) {
	m0 := NewWithHasher[int, string](testIntHasher{})
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
	m := NewWithHasher[uint64, string](identityUint64Hasher{})
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
	m := NewWithHasher[uint64, string](identityUint64Hasher{})
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
	m := NewWithHasher[int, string](constantIntHasher{})
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
	m := NewWithHasher[int, string](splitCollisionHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")

	assertMissing[int, string](t, m, 3)
	validateMap(t, m)
}

func TestMapCollisionDeleteMissingReturnsEquivalentMap(t *testing.T) {
	m := NewWithHasher[int, string](constantIntHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")
	n := m.Delete(3)

	assertLen(t, n, 2)
	assertGet(t, n, 1, "one")
	assertGet(t, n, 2, "two")
	validateMap(t, n)
}

func TestMapCollisionDeleteHashMismatchReturnsEquivalentMap(t *testing.T) {
	m := NewWithHasher[int, string](splitCollisionHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")
	n := m.Delete(3)

	assertLen(t, n, 2)
	assertGet(t, n, 1, "one")
	assertGet(t, n, 2, "two")
	validateMap(t, n)
}

func TestMapCollisionInsertDoesNotMutateSnapshot(t *testing.T) {
	m := NewWithHasher[int, string](constantIntHasher{})
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
	b := NewBuilderWithHasher[int, string](constantIntHasher{})
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
	m := NewWithHasher[int, string](constantIntHasher{})
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
	m := NewWithHasher[int, string](constantIntHasher{})
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
	m := NewWithHasher[int, string](splitCollisionHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")
	m = m.Set(3, "three")

	assertLen(t, m, 3)
	assertGet(t, m, 1, "one")
	assertGet(t, m, 2, "two")
	assertGet(t, m, 3, "three")
	validateMap(t, m)
}

func TestMapCollisionExpansionDoesNotMutateSnapshot(t *testing.T) {
	m := NewWithHasher[int, string](splitCollisionHasher{})
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

func TestMapCollisionExpansionDeepDivergence(t *testing.T) {
	m := NewWithHasher[int, string](splitCollisionHasher{})
	m = m.Set(1, "one")
	m = m.Set(2, "two")

	n := m.Set(4, "four")

	assertLen(t, m, 2)
	assertMissing[int, string](t, m, 4)
	assertLen(t, n, 3)
	assertGet(t, n, 1, "one")
	assertGet(t, n, 2, "two")
	assertGet(t, n, 4, "four")
	validateMap(t, m)
	validateMap(t, n)
}

func TestMapDeleteCanonicalizesSingletonCollision(t *testing.T) {
	m := NewWithHasher[int, string](constantIntHasher{})
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

// corruptSingletonCollisionRoot builds a tree whose collision child holds a
// single entry — a state no reachable operation produces, since collision
// nodes are created with two entries and collapse through singleton() before
// dropping below that.
func corruptSingletonCollisionRoot() *node[int, string] {
	corrupt := &node[int, string]{collision: &collisionNode[int, string]{
		hash:    1,
		entries: []collisionEntry[int, string]{{key: 1, value: "one"}},
	}}
	return &node[int, string]{nodeMap: bitpos(fragment(1, 0)), children: []*node[int, string]{corrupt}}
}

func TestDeleteUndersizedCollisionChildPanics(t *testing.T) {
	root := corruptSingletonCollisionRoot()
	assertPanics(t, func() { root.delete(1, 1, 0, constantIntHasher{}) })
}

func TestMapRange(t *testing.T) {
	m := NewWithHasher[int, string](testIntHasher{})
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
	m := NewWithHasher[int, int](testIntHasher{})
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
	m := NewWithHasher[int, string](constantIntHasher{})
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
	m := NewWithHasher[int, int](constantIntHasher{})
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

func TestMapConcurrentReaders(t *testing.T) {
	m := NewWithHasher[int, int](collisionProneHasher{})
	for i := 0; i < 200; i++ {
		m = m.Set(i, i)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 200; i++ {
				if v, ok := m.Get(i); !ok || v != i {
					t.Errorf("Get(%d) = %d, %v; want %d, true", i, v, ok, i)
					return
				}
			}
			count := 0
			m.Range(func(int, int) bool {
				count++
				return true
			})
			if count != 200 {
				t.Errorf("range count = %d, want 200", count)
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestMapIterationDeterministicForSameHistory(t *testing.T) {
	// Seeding a same-bucket pair first forms a shallow collision node, so the
	// next different-bucket insert pushes it down a branch chain.
	buildMap := func() Map[int, int] {
		m := NewWithHasher[int, int](collisionProneHasher{})
		m = m.Set(0, 0)
		m = m.Set(7, 7)
		for i := 0; i < 60; i++ {
			m = m.Set(i, i)
		}
		for i := 0; i < 60; i += 3 {
			m = m.Delete(i)
		}
		return m
	}
	buildBuilder := func() Map[int, int] {
		b := NewBuilderWithHasher[int, int](collisionProneHasher{})
		b.Set(0, 0)
		b.Set(7, 7)
		for i := 0; i < 60; i++ {
			b.Set(i, i)
		}
		for i := 0; i < 60; i += 3 {
			b.Delete(i)
		}
		return b.Map()
	}

	assertSameOrder(t, rangeKeys(buildMap()), rangeKeys(buildMap()))
	assertSameOrder(t, rangeKeys(buildBuilder()), rangeKeys(buildBuilder()))
}

func rangeKeys[K comparable, V any](m Map[K, V]) []K {
	var keys []K
	m.Range(func(k K, _ V) bool {
		keys = append(keys, k)
		return true
	})
	return keys
}

func assertSameOrder[K comparable](t *testing.T, a, b []K) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("iteration lengths differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("iteration order differs at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestNewUsesLanguageEquality(t *testing.T) {
	type point struct{ x, y int }
	m := New[point, string]()
	m = m.Set(point{1, 2}, "a")
	m = m.Set(point{1, 2}, "b")
	m = m.Set(point{2, 1}, "c")

	assertLen(t, m, 2)
	assertGet(t, m, point{1, 2}, "b")
	assertGet(t, m, point{2, 1}, "c")
	n := m.Delete(point{1, 2})
	assertLen(t, n, 1)
	assertMissing[point, string](t, n, point{1, 2})
	validateMap(t, m)
	validateMap(t, n)
}

func TestNewNaNKeysAreInsertOnly(t *testing.T) {
	// NaN keys mirror the builtin map anomaly: never == to anything, so they
	// can be inserted and iterated but not looked up or deleted. A split
	// re-places such an entry under a freshly computed hash spliced onto the
	// shared path bits, so the map stays structurally sound for every other
	// key; the validator is skipped because rehashing a NaN key cannot
	// reproduce its placement.
	nan := math.NaN()
	m := New[float64, string]()
	m = m.Set(nan, "a")
	m = m.Set(nan, "b")

	assertLen(t, m, 2)
	assertMissing[float64, string](t, m, nan)
	n := m.Delete(nan)
	assertLen(t, n, 2)

	visits := 0
	m.Range(func(float64, string) bool {
		visits++
		return true
	})
	if visits != 2 {
		t.Fatalf("range visits = %d, want 2", visits)
	}
}

// unstableIntHasher deterministically reproduces the default hasher's NaN
// anomaly: key 0 is never equal to anything, itself included, and hashes
// differently on every call, with the variation confined to bits below the
// first diverging fragment. Every other key hashes to 0.
type unstableIntHasher struct{ calls uint64 }

func (h *unstableIntHasher) Hash(v int) uint64 {
	if v != 0 {
		return 0
	}
	h.calls++
	return h.calls - 1
}

func (h *unstableIntHasher) Equal(a, b int) bool { return a != 0 && b != 0 && a == b }

func TestMapSetSplitOfUnstableHashKeyTerminates(t *testing.T) {
	// A stored key may rehash at split time to a value that disagrees with
	// the incoming hash only below the level where the split happens; the
	// split must still terminate and keep both entries reachable by
	// structure. The validator is skipped for the same reason as
	// TestNewNaNKeysAreInsertOnly.
	m := NewWithHasher[int, string](&unstableIntHasher{})
	m = m.Set(0, "unstable")
	m = m.Set(1, "stable")

	assertLen(t, m, 2)
	assertGet(t, m, 1, "stable")
	assertRangeVisits(t, m, 2)
}

func TestBuilderSetSplitOfUnstableHashKeyTerminates(t *testing.T) {
	b := NewBuilderWithHasher[int, string](&unstableIntHasher{})
	b.Set(0, "unstable")
	b.Set(1, "stable")
	m := b.Map()

	assertLen(t, m, 2)
	assertGet(t, m, 1, "stable")
	assertRangeVisits(t, m, 2)
}

func TestNewNonComparableDynamicKeyPanics(t *testing.T) {
	m := New[any, int]()
	assertPanics(t, func() { m.Set([]int{1}, 1) })
}

func TestMapQuickAgainstBuiltin(t *testing.T) {
	quickCheckMap(t, testIntHasher{})
}

func TestMapQuickAgainstBuiltinDefaultHasher(t *testing.T) {
	quickCheckMap(t, defaultHasher[int]{})
}

func TestMapQuickAgainstBuiltinCollisionProne(t *testing.T) {
	quickCheckMap(t, collisionProneHasher{})
}

func quickCheckMap(t *testing.T, hasher Hasher[int]) {
	t.Helper()
	prop := func(ops []uint64) bool {
		m := NewWithHasher[int, int](hasher)
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

func TestMapQuickDerivedMutationsDoNotChangeSource(t *testing.T) {
	quickCheckSnapshotUnchanged(t, testIntHasher{})
}

func TestMapQuickDerivedMutationsDoNotChangeSourceDefaultHasher(t *testing.T) {
	quickCheckSnapshotUnchanged(t, defaultHasher[int]{})
}

func TestMapQuickDerivedMutationsDoNotChangeSourceCollisionProne(t *testing.T) {
	quickCheckSnapshotUnchanged(t, collisionProneHasher{})
}

// quickCheckSnapshotUnchanged pins the structural-sharing contract: maps
// returned by Set and Delete share structure, and no later derived mutation
// may change the reachable contents or internal structure of any previously
// returned map — neither the seed snapshot nor any intermediate derivation.
func quickCheckSnapshotUnchanged(t *testing.T, hasher Hasher[int]) {
	t.Helper()
	type snapshot struct {
		m    Map[int, int]
		want map[int]int
	}
	prop := func(seedOps, deriveOps []uint64) bool {
		src := NewWithHasher[int, int](hasher)
		want := map[int]int{}
		for _, op := range seedOps {
			key := int((op >> 2) % 97)
			value := int(op >> 9)
			if op&3 == 3 {
				src = src.Delete(key)
				delete(want, key)
			} else {
				src = src.Set(key, value)
				want[key] = value
			}
		}

		// Derive maps from the snapshot and from earlier derivations; after
		// each derivation every previously returned map must be unchanged.
		snapshots := []snapshot{{m: src, want: maps.Clone(want)}}
		for _, op := range deriveOps {
			key := int((op >> 2) % 97)
			value := int(op >> 9)
			base := snapshots[len(snapshots)-1]
			if op&1 == 0 {
				base = snapshots[0]
			}
			derivedWant := maps.Clone(base.want)
			if derivedWant == nil {
				derivedWant = map[int]int{}
			}
			var derived Map[int, int]
			if op&2 == 0 {
				derived = base.m.Set(key, value)
				derivedWant[key] = value
			} else {
				derived = base.m.Delete(key)
				delete(derivedWant, key)
			}
			snapshots = append(snapshots, snapshot{m: derived, want: derivedWant})
			for _, s := range snapshots {
				if !mapsEqual(s.m, s.want) {
					t.Logf("snapshot contents changed")
					return false
				}
				if err := checkMap(s.m); err != nil {
					t.Logf("snapshot structure invalid: %v", err)
					return false
				}
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
		_ = NewWithHasher[int, int](nil).Set(1, 1)
	})
}

func assertRangeVisits[K, V any](t *testing.T, m Map[K, V], want int) {
	t.Helper()
	visits := 0
	m.Range(func(K, V) bool {
		visits++
		return true
	})
	if visits != want {
		t.Fatalf("range visits = %d, want %d", visits, want)
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
	count, err := checkNode(m.root, 0, 0, m.hasher)
	if err != nil {
		return err
	}
	if count != m.Len() {
		return fmt.Errorf("reachable count = %d, len = %d", count, m.Len())
	}
	return nil
}

// checkNode validates the subtree rooted at n. prefix holds the hash bits
// implied by the path from the root: every hash stored below n must match
// it in the low shift bits, otherwise lookups walking those fragments
// cannot reach the entry.
func checkNode[K comparable, V comparable](n *node[K, V], shift uint, prefix uint64, h Hasher[K]) (int, error) {
	if n == nil {
		return 0, fmt.Errorf("nil child")
	}
	prefixMask := uint64(1)<<shift - 1
	if n.isCollision() {
		c := n.collision
		if n.dataMap != 0 || n.nodeMap != 0 || n.entries != nil || n.children != nil {
			return 0, fmt.Errorf("collision node carries branch payload")
		}
		if len(c.entries) < 2 {
			return 0, fmt.Errorf("collision len = %d, want >= 2", len(c.entries))
		}
		if c.hash&prefixMask != prefix {
			return 0, fmt.Errorf("collision node stored under wrong path")
		}
		seen := map[K]struct{}{}
		for _, e := range c.entries {
			if h.Hash(e.key) != c.hash {
				return 0, fmt.Errorf("stored hash mismatch")
			}
			if _, ok := seen[e.key]; ok {
				return 0, fmt.Errorf("duplicate collision key")
			}
			seen[e.key] = struct{}{}
		}
		return len(c.entries), nil
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
	// The shift-60 fragment has only 4 hash bits, so fragments >= 16 are
	// unreachable there.
	if shift == 60 && (n.dataMap|n.nodeMap) >= 1<<16 {
		return 0, fmt.Errorf("fragment beyond hash range")
	}

	count := 0
	entryIdx := 0
	for frag := uint(0); frag < 32; frag++ {
		bit := uint32(1) << frag
		if n.dataMap&bit == 0 {
			continue
		}
		e := n.entries[entryIdx]
		entryIdx++
		hash := h.Hash(e.key)
		if bitpos(fragment(hash, shift)) != bit {
			return 0, fmt.Errorf("entry stored under wrong fragment")
		}
		if hash&prefixMask != prefix {
			return 0, fmt.Errorf("entry stored under wrong path")
		}
		count++
	}

	childIdx := 0
	for frag := uint(0); frag < 32; frag++ {
		bit := uint32(1) << frag
		if n.nodeMap&bit == 0 {
			continue
		}
		child := n.children[childIdx]
		childIdx++
		childCount, err := checkNode(child, shift+fragmentBits, prefix|uint64(frag)<<shift, h)
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
