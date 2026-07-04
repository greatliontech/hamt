package hamt

import "testing"

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
	case 4:
		return 1 << (3 * fragmentBits)
	default:
		return uint64(v) << fragmentBits
	}
}

func (splitCollisionHasher) Equal(a, b int) bool { return a == b }

// collisionProneHasher maps keys into seven buckets whose hashes agree on
// the six lowest fragments, so property tests exercise full collisions,
// collision-node splits, and multi-level branch chains together.
type collisionProneHasher struct{}

func (collisionProneHasher) Hash(v int) uint64   { return uint64(v%7) << 32 }
func (collisionProneHasher) Equal(a, b int) bool { return a == b }

type identityUint64Hasher struct{}

func (identityUint64Hasher) Hash(v uint64) uint64   { return v }
func (identityUint64Hasher) Equal(a, b uint64) bool { return a == b }
