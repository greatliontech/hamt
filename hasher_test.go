package hamt

import "testing"

func TestDefaultHasherLanguageEquality(t *testing.T) {
	h := defaultHasher[string]{}
	first := h.Hash("jane")
	second := h.Hash("jane")
	if first != second {
		t.Fatal("Hash is not stable within the process")
	}
	if !h.Equal("jane", "jane") {
		t.Fatal(`Equal("jane", "jane") = false, want true`)
	}
	if h.Equal("jane", "susy") {
		t.Fatal(`Equal("jane", "susy") = true, want false`)
	}
}

// testIntHasher is a deterministic, well-distributed hasher so test
// failures reproduce across runs, unlike the process-seeded default.
type testIntHasher struct{}

func (testIntHasher) Hash(v int) uint64   { return mix64(uint64(v)) }
func (testIntHasher) Equal(a, b int) bool { return a == b }

// mix64 is the splitmix64 finalizer.
func mix64(v uint64) uint64 {
	v += 0x9e3779b97f4a7c15
	v = (v ^ (v >> 30)) * 0xbf58476d1ce4e5b9
	v = (v ^ (v >> 27)) * 0x94d049bb133111eb
	return v ^ (v >> 31)
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
