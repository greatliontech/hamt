package hamt

import (
	"fmt"
	"testing"
)

var (
	benchValueSink int
	benchBoolSink  bool
	benchMapSink   Map[benchKey, int]
)

func BenchmarkMapGetHit(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size + 1)
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			m := buildOurs(keys[:size])
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v, ok := m.Get(keys[i%size])
				benchValueSink = v
				benchBoolSink = ok
			}
		})
	}
}

func BenchmarkMapSetInsert(b *testing.B) {
	for _, size := range benchSizes() {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			keys := benchKeys(size + 1)
			m := buildOurs(keys[:size])
			missing := keys[size]
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchMapSink = m.Set(missing, i)
			}
		})
	}
}

func BenchmarkMapSetOverwrite(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			m := buildOurs(keys)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchMapSink = m.Set(keys[i%size], i)
			}
		})
	}
}

func BenchmarkMapDeleteHit(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			m := buildOurs(keys)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchMapSink = m.Delete(keys[i%size])
			}
		})
	}
}

func BenchmarkMapBuild(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchMapSink = buildOurs(keys)
			}
		})
	}
}

func BenchmarkMapBuilderBuild(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchMapSink = buildOursBuilder(keys)
			}
		})
	}
}

func BenchmarkMapRange(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			m := buildOurs(keys)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sum := 0
				m.Range(func(_ benchKey, v int) bool {
					sum += v
					return true
				})
				benchValueSink = sum
			}
		})
	}
}

func BenchmarkMapFullCollisionGetHit(b *testing.B) {
	benchmarkGetHit(b, collisionBenchSizes(), collisionHasher{})
}

func BenchmarkMapFullCollisionSetInsert(b *testing.B) {
	benchmarkSetInsert(b, collisionBenchSizes(), collisionHasher{})
}

func BenchmarkMapFullCollisionBuild(b *testing.B) {
	benchmarkBuild(b, collisionBenchSizes(), collisionHasher{})
}

func BenchmarkMapFullCollisionBuilderBuild(b *testing.B) {
	benchmarkBuilderBuild(b, collisionBenchSizes(), collisionHasher{})
}

func BenchmarkMapCollisionSplitSetInsert(b *testing.B) {
	for _, size := range collisionBenchSizes() {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			keys := benchKeys(size + 1)
			missing := keys[size]
			hasher := markedCollisionHasher{marked: missing}
			m := buildOursWithHasher(keys[:size], hasher)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchMapSink = m.Set(missing, i)
			}
		})
	}
}

func BenchmarkMapSharedPrefixGetHit(b *testing.B) {
	benchmarkGetHit(b, benchSizes(), sharedPrefixHasher{})
}

func BenchmarkMapSharedPrefixBuild(b *testing.B) {
	benchmarkBuild(b, benchSizes(), sharedPrefixHasher{})
}

func benchmarkGetHit(b *testing.B, sizes []int, hasher Hasher[benchKey]) {
	for _, size := range sizes {
		keys := benchKeys(size + 1)
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			m := buildOursWithHasher(keys[:size], hasher)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v, ok := m.Get(keys[i%size])
				benchValueSink = v
				benchBoolSink = ok
			}
		})
	}
}

func benchmarkSetInsert(b *testing.B, sizes []int, hasher Hasher[benchKey]) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			keys := benchKeys(size + 1)
			m := buildOursWithHasher(keys[:size], hasher)
			missing := keys[size]
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchMapSink = m.Set(missing, i)
			}
		})
	}
}

func benchmarkBuild(b *testing.B, sizes []int, hasher Hasher[benchKey]) {
	for _, size := range sizes {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchMapSink = buildOursWithHasher(keys, hasher)
			}
		})
	}
}

func benchmarkBuilderBuild(b *testing.B, sizes []int, hasher Hasher[benchKey]) {
	for _, size := range sizes {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchMapSink = buildOursBuilderWithHasher(keys, hasher)
			}
		})
	}
}

func benchSizes() []int {
	return []int{1, 8, 32, 1024}
}

func collisionBenchSizes() []int {
	return []int{8, 32, 256}
}

func benchKeys(n int) []benchKey {
	keys := make([]benchKey, n)
	for i := range keys {
		keys[i] = benchKey(i + 1)
	}
	return keys
}

func buildOurs(keys []benchKey) Map[benchKey, int] {
	return buildOursWithHasher(keys, benchHasher{})
}

func buildOursWithHasher(keys []benchKey, hasher Hasher[benchKey]) Map[benchKey, int] {
	m := NewWithHasher[benchKey, int](hasher)
	for _, key := range keys {
		m = m.Set(key, int(key))
	}
	return m
}

func buildOursBuilder(keys []benchKey) Map[benchKey, int] {
	return buildOursBuilderWithHasher(keys, benchHasher{})
}

func buildOursBuilderWithHasher(keys []benchKey, hasher Hasher[benchKey]) Map[benchKey, int] {
	b := NewBuilderWithHasher[benchKey, int](hasher)
	for _, key := range keys {
		b.Set(key, int(key))
	}
	return b.Map()
}

type benchKey uint64

type benchHasher struct{}

func (benchHasher) Hash(k benchKey) uint64   { return mix64(uint64(k)) }
func (benchHasher) Equal(a, b benchKey) bool { return a == b }

type collisionHasher struct{}

func (collisionHasher) Hash(benchKey) uint64     { return 1 }
func (collisionHasher) Equal(a, b benchKey) bool { return a == b }

// markedCollisionHasher collides every key on hash 0 except marked, whose
// hash agrees with 0 on the three lowest fragments and then diverges, so
// inserting marked splits an existing collision node.
type markedCollisionHasher struct{ marked benchKey }

func (h markedCollisionHasher) Hash(k benchKey) uint64 {
	if k == h.marked {
		return 1 << (3 * fragmentBits)
	}
	return 0
}

func (h markedCollisionHasher) Equal(a, b benchKey) bool { return a == b }

type sharedPrefixHasher struct{}

func (sharedPrefixHasher) Hash(k benchKey) uint64   { return uint64(k) << fragmentBits }
func (sharedPrefixHasher) Equal(a, b benchKey) bool { return a == b }
