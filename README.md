# hamt

`hamt` is a generic immutable hash map for Go.

The map is persistent: `Set` and `Delete` return new maps while previous maps
remain valid snapshots that may share structure. A published map is safe for
concurrent readers as long as callers do not mutate data reachable through
keys or values.

## Installation

```bash
go get github.com/greatliontech/hamt
```

## Usage

`New` keys the map by language equality: the key type must be `comparable`,
keys are compared with `==`, and hashes come from a hash function seeded once
per process.

```go
m := hamt.New[string, int]()
m = m.Set("jane", 100)

v, ok := m.Get("jane")
fmt.Println(v, ok) // 100 true

n := m.Delete("jane")
fmt.Println(n.Len(), m.Len()) // 0 1

m.Range(func(k string, v int) bool {
 fmt.Println(k, v)
 return true
})
```

`NewWithHasher` takes a `Hasher[K]` instead, for custom key identity or for
key types that are not comparable:

```go
m := hamt.NewWithHasher[string, string](caseInsensitiveHasher{})
```

A `Hasher[K]` defines what "the same key" means: `Equal` must be an
equivalence relation, equal keys must hash equally, and hashes must be stable
while a key is stored.

For bulk construction, use a builder. It mutates private state, so building
avoids the per-operation copying of the persistent path. `Map` returns the
built immutable map and invalidates the builder. `NewBuilder` and
`NewBuilderWithHasher` mirror the map constructors:

```go
b := hamt.NewBuilder[string, int]()
b.Set("jane", 100)
b.Set("susy", 200)
m := b.Map()
```
