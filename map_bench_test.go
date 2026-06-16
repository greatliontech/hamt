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

func benchSizes() []int {
	return []int{1, 8, 32, 1024}
}

func benchKeys(n int) []benchKey {
	keys := make([]benchKey, n)
	for i := range keys {
		keys[i] = benchKey(i + 1)
	}
	return keys
}

func buildOurs(keys []benchKey) Map[benchKey, int] {
	m := NewMap[benchKey, int](benchHasher{})
	for _, key := range keys {
		m = m.Set(key, int(key))
	}
	return m
}

func buildBen(keys []benchKey) *ben.Map[benchKey, int] {
	m := ben.NewMap[benchKey, int](benBenchHasher{})
	for _, key := range keys {
		m = m.Set(key, int(key))
	}
	return m
}

func buildOursBuilder(keys []benchKey) Map[benchKey, int] {
	b := NewBuilder[benchKey, int](benchHasher{})
	for _, key := range keys {
		b.Set(key, int(key))
	}
	return b.Map()
}

func buildBenBuilder(keys []benchKey) *ben.Map[benchKey, int] {
	b := ben.NewMapBuilder[benchKey, int](benBenchHasher{})
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
