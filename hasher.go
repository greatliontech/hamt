package hamt

import "hash/maphash"

// Hasher hashes keys and checks them for equality.
type Hasher[K any] interface {
	Hash(K) uint64
	Equal(K, K) bool
}

// defaultSeed is created once per process so every default-keyed map hashes
// a self-equal key identically for the process lifetime, as the Hasher
// stability contract requires. Keys not equal to themselves (containing NaN)
// hash unpredictably on each call, matching the builtin map's insert-only
// anomaly for such keys. Hashes intentionally differ between processes.
var defaultSeed = maphash.MakeSeed()

// defaultHasher keys a map by language equality: Equal is == and Hash is a
// process-seeded maphash over the key's value.
type defaultHasher[K comparable] struct{}

func (defaultHasher[K]) Hash(v K) uint64   { return maphash.Comparable(defaultSeed, v) }
func (defaultHasher[K]) Equal(a, b K) bool { return a == b }
