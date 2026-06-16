package hamt

import (
	"fmt"
	"testing"

	ben "github.com/benbjohnson/immutable"
)

var (
	benchValueSink int
	benchBoolSink  bool
	benchMapSink   Map[benchKey, int]
	benchBenSink   *ben.Map[benchKey, int]
)

func BenchmarkMapGetHit(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size + 1)
		b.Run(fmt.Sprintf("ours/%d", size), func(b *testing.B) {
			m := buildOurs(keys[:size])
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v, ok := m.Get(keys[i%size])
				benchValueSink = v
				benchBoolSink = ok
			}
		})
		b.Run(fmt.Sprintf("benbjohnson/%d", size), func(b *testing.B) {
			m := buildBen(keys[:size])
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
		b.Run(fmt.Sprintf("ours/%d", size), func(b *testing.B) {
			keys := benchKeys(size + 1)
			m := buildOurs(keys[:size])
			missing := keys[size]
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchMapSink = m.Set(missing, i)
			}
		})
		b.Run(fmt.Sprintf("benbjohnson/%d", size), func(b *testing.B) {
			keys := benchKeys(size + 1)
			m := buildBen(keys[:size])
			missing := keys[size]
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchBenSink = m.Set(missing, i)
			}
		})
	}
}

func BenchmarkMapSetOverwrite(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("ours/%d", size), func(b *testing.B) {
			m := buildOurs(keys)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchMapSink = m.Set(keys[i%size], i)
			}
		})
		b.Run(fmt.Sprintf("benbjohnson/%d", size), func(b *testing.B) {
			m := buildBen(keys)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchBenSink = m.Set(keys[i%size], i)
			}
		})
	}
}

func BenchmarkMapDeleteHit(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("ours/%d", size), func(b *testing.B) {
			m := buildOurs(keys)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchMapSink = m.Delete(keys[i%size])
			}
		})
		b.Run(fmt.Sprintf("benbjohnson/%d", size), func(b *testing.B) {
			m := buildBen(keys)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchBenSink = m.Delete(keys[i%size])
			}
		})
	}
}

func BenchmarkMapBuild(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("ours/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchMapSink = buildOurs(keys)
			}
		})
		b.Run(fmt.Sprintf("benbjohnson/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchBenSink = buildBen(keys)
			}
		})
	}
}

func BenchmarkMapBuilderBuild(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("ours/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchMapSink = buildOursBuilder(keys)
			}
		})
		b.Run(fmt.Sprintf("benbjohnson/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchBenSink = buildBenBuilder(keys)
			}
		})
	}
}

func BenchmarkMapRange(b *testing.B) {
	for _, size := range benchSizes() {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("ours/%d", size), func(b *testing.B) {
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
		b.Run(fmt.Sprintf("benbjohnson/%d", size), func(b *testing.B) {
			m := buildBen(keys)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sum := 0
				itr := m.Iterator()
				for !itr.Done() {
					_, v, _ := itr.Next()
					sum += v
				}
				benchValueSink = sum
			}
		})
	}
}

func BenchmarkMapFullCollisionGetHit(b *testing.B) {
	benchmarkGetHit(b, collisionBenchSizes(), collisionHasher{}, benCollisionHasher{})
}

func BenchmarkMapFullCollisionSetInsert(b *testing.B) {
	benchmarkSetInsert(b, collisionBenchSizes(), collisionHasher{}, benCollisionHasher{})
}

func BenchmarkMapFullCollisionBuild(b *testing.B) {
	benchmarkBuild(b, collisionBenchSizes(), collisionHasher{}, benCollisionHasher{})
}

func BenchmarkMapFullCollisionBuilderBuild(b *testing.B) {
	benchmarkBuilderBuild(b, collisionBenchSizes(), collisionHasher{}, benCollisionHasher{})
}

func BenchmarkMapSharedPrefixGetHit(b *testing.B) {
	benchmarkGetHit(b, benchSizes(), sharedPrefixHasher{}, benSharedPrefixHasher{})
}

func BenchmarkMapSharedPrefixBuild(b *testing.B) {
	benchmarkBuild(b, benchSizes(), sharedPrefixHasher{}, benSharedPrefixHasher{})
}

func benchmarkGetHit(b *testing.B, sizes []int, oursHasher Hasher[benchKey], benHasher ben.Hasher[benchKey]) {
	for _, size := range sizes {
		keys := benchKeys(size + 1)
		b.Run(fmt.Sprintf("ours/%d", size), func(b *testing.B) {
			m := buildOursWithHasher(keys[:size], oursHasher)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v, ok := m.Get(keys[i%size])
				benchValueSink = v
				benchBoolSink = ok
			}
		})
		b.Run(fmt.Sprintf("benbjohnson/%d", size), func(b *testing.B) {
			m := buildBenWithHasher(keys[:size], benHasher)
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

func benchmarkSetInsert(b *testing.B, sizes []int, oursHasher Hasher[benchKey], benHasher ben.Hasher[benchKey]) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("ours/%d", size), func(b *testing.B) {
			keys := benchKeys(size + 1)
			m := buildOursWithHasher(keys[:size], oursHasher)
			missing := keys[size]
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchMapSink = m.Set(missing, i)
			}
		})
		b.Run(fmt.Sprintf("benbjohnson/%d", size), func(b *testing.B) {
			keys := benchKeys(size + 1)
			m := buildBenWithHasher(keys[:size], benHasher)
			missing := keys[size]
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchBenSink = m.Set(missing, i)
			}
		})
	}
}

func benchmarkBuild(b *testing.B, sizes []int, oursHasher Hasher[benchKey], benHasher ben.Hasher[benchKey]) {
	for _, size := range sizes {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("ours/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchMapSink = buildOursWithHasher(keys, oursHasher)
			}
		})
		b.Run(fmt.Sprintf("benbjohnson/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchBenSink = buildBenWithHasher(keys, benHasher)
			}
		})
	}
}

func benchmarkBuilderBuild(b *testing.B, sizes []int, oursHasher Hasher[benchKey], benHasher ben.Hasher[benchKey]) {
	for _, size := range sizes {
		keys := benchKeys(size)
		b.Run(fmt.Sprintf("ours/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchMapSink = buildOursBuilderWithHasher(keys, oursHasher)
			}
		})
		b.Run(fmt.Sprintf("benbjohnson/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchBenSink = buildBenBuilderWithHasher(keys, benHasher)
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
	m := NewMap[benchKey, int](hasher)
	for _, key := range keys {
		m = m.Set(key, int(key))
	}
	return m
}

func buildBen(keys []benchKey) *ben.Map[benchKey, int] {
	return buildBenWithHasher(keys, benBenchHasher{})
}

func buildBenWithHasher(keys []benchKey, hasher ben.Hasher[benchKey]) *ben.Map[benchKey, int] {
	m := ben.NewMap[benchKey, int](hasher)
	for _, key := range keys {
		m = m.Set(key, int(key))
	}
	return m
}

func buildOursBuilder(keys []benchKey) Map[benchKey, int] {
	return buildOursBuilderWithHasher(keys, benchHasher{})
}

func buildOursBuilderWithHasher(keys []benchKey, hasher Hasher[benchKey]) Map[benchKey, int] {
	b := NewBuilder[benchKey, int](hasher)
	for _, key := range keys {
		b.Set(key, int(key))
	}
	return b.Map()
}

func buildBenBuilder(keys []benchKey) *ben.Map[benchKey, int] {
	return buildBenBuilderWithHasher(keys, benBenchHasher{})
}

func buildBenBuilderWithHasher(keys []benchKey, hasher ben.Hasher[benchKey]) *ben.Map[benchKey, int] {
	b := ben.NewMapBuilder[benchKey, int](hasher)
	for _, key := range keys {
		b.Set(key, int(key))
	}
	return b.Map()
}

type benchKey uint64

type benchHasher struct{}

func (benchHasher) Hash(k benchKey) uint64   { return mix64(uint64(k)) }
func (benchHasher) Equal(a, b benchKey) bool { return a == b }

type benBenchHasher struct{}

func (benBenchHasher) Hash(k benchKey) uint32   { return uint32(mix64(uint64(k))) }
func (benBenchHasher) Equal(a, b benchKey) bool { return a == b }

type collisionHasher struct{}

func (collisionHasher) Hash(benchKey) uint64     { return 1 }
func (collisionHasher) Equal(a, b benchKey) bool { return a == b }

type benCollisionHasher struct{}

func (benCollisionHasher) Hash(benchKey) uint32     { return 1 }
func (benCollisionHasher) Equal(a, b benchKey) bool { return a == b }

type sharedPrefixHasher struct{}

func (sharedPrefixHasher) Hash(k benchKey) uint64   { return uint64(k) << fragmentBits }
func (sharedPrefixHasher) Equal(a, b benchKey) bool { return a == b }

type benSharedPrefixHasher struct{}

func (benSharedPrefixHasher) Hash(k benchKey) uint32 {
	return uint32(uint64(k) << fragmentBits)
}

func (benSharedPrefixHasher) Equal(a, b benchKey) bool { return a == b }
