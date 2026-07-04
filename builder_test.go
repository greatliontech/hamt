package hamt

import (
	"fmt"
	"testing"
	"testing/quick"
)

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

func TestBuilderCollisionExpansionDeepDivergence(t *testing.T) {
	b := NewBuilder[int, string](splitCollisionHasher{})
	b.Set(1, "one")
	b.Set(2, "two")
	b.Set(4, "four")
	m := b.Map()

	assertLen(t, m, 3)
	assertGet(t, m, 1, "one")
	assertGet(t, m, 2, "two")
	assertGet(t, m, 4, "four")
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

func TestDeleteMutableUndersizedCollisionChildPanics(t *testing.T) {
	root := corruptSingletonCollisionRoot()
	assertPanics(t, func() { root.deleteMutable(1, 1, 0, constantIntHasher{}) })
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
	quickCheckBuilder(t, IntHasher{})
}

func TestBuilderQuickAgainstBuiltinCollisionProne(t *testing.T) {
	quickCheckBuilder(t, collisionProneHasher{})
}

func quickCheckBuilder(t *testing.T, hasher Hasher[int]) {
	t.Helper()
	prop := func(ops []uint64) bool {
		b := NewBuilder[int, int](hasher)
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
