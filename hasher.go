package hamt

// Hasher hashes keys and checks them for equality.
type Hasher[K any] interface {
	Hash(K) uint64
	Equal(K, K) bool
}

// IntHasher hashes int keys.
type IntHasher struct{}

func (IntHasher) Hash(v int) uint64   { return mix64(uint64(v)) }
func (IntHasher) Equal(a, b int) bool { return a == b }

// Uint64Hasher hashes uint64 keys.
type Uint64Hasher struct{}

func (Uint64Hasher) Hash(v uint64) uint64   { return mix64(v) }
func (Uint64Hasher) Equal(a, b uint64) bool { return a == b }

// StringHasher hashes string keys.
type StringHasher struct{}

func (StringHasher) Hash(s string) uint64 {
	const prime = 1099511628211
	var hash uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= prime
	}
	return hash
}

func (StringHasher) Equal(a, b string) bool { return a == b }

func mix64(v uint64) uint64 {
	v += 0x9e3779b97f4a7c15
	v = (v ^ (v >> 30)) * 0xbf58476d1ce4e5b9
	v = (v ^ (v >> 27)) * 0x94d049bb133111eb
	return v ^ (v >> 31)
}
