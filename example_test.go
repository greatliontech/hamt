package hamt_test

import (
	"fmt"
	"hash/maphash"
	"strings"

	"github.com/greatliontech/hamt"
)

func ExampleMap() {
	m0 := hamt.New[string, int]()
	m1 := m0.Set("jane", 100)
	m2 := m1.Set("jane", 300)

	v1, _ := m1.Get("jane")
	v2, _ := m2.Get("jane")
	_, ok := m0.Get("jane")

	fmt.Println(v1)
	fmt.Println(v2)
	fmt.Println(ok)

	// Output:
	// 100
	// 300
	// false
}

func ExampleBuilder() {
	b := hamt.NewBuilder[string, int]()
	b.Set("jane", 100)
	b.Set("susy", 200)
	b.Set("jane", 300)
	b.Delete("susy")

	m := b.Map()
	v, ok := m.Get("jane")
	_, deleted := m.Get("susy")

	fmt.Println(m.Len())
	fmt.Println(v, ok)
	fmt.Println(deleted)

	// Output:
	// 1
	// 300 true
	// false
}

func ExampleHasher() {
	m := hamt.NewWithHasher[string, string](caseInsensitiveHasher{})
	m = m.Set("Jane", "first")
	m = m.Set("jane", "second")

	v, _ := m.Get("JANE")
	fmt.Println(m.Len())
	fmt.Println(v)

	// Output:
	// 1
	// second
}

var caseInsensitiveSeed = maphash.MakeSeed()

type caseInsensitiveHasher struct{}

func (caseInsensitiveHasher) Hash(s string) uint64 {
	return maphash.String(caseInsensitiveSeed, strings.ToLower(s))
}

func (caseInsensitiveHasher) Equal(a, b string) bool {
	return strings.ToLower(a) == strings.ToLower(b)
}
