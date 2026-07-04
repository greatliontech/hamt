# hamt

`hamt` is a generic immutable hash map for Go.

The map is persistent: `Set` and `Delete` return new maps while previous maps
remain valid snapshots that may share structure. A published map is safe for
concurrent readers as long as callers do not mutate data reachable through
keys or values. Keys are compared and hashed through a caller-provided
`Hasher[K]`; `IntHasher`, `Uint64Hasher`, and `StringHasher` are provided.

```go
m := hamt.NewMap[string, int](hamt.StringHasher{})
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

For bulk construction, use a builder. It mutates private state, so building
avoids the per-operation copying of the persistent path; `Map` returns the
built immutable map and invalidates the builder:

```go
b := hamt.NewBuilder[string, int](hamt.StringHasher{})
b.Set("jane", 100)
b.Set("susy", 200)
m := b.Map()
```

Custom key types implement `Hasher[K]`: `Equal` must be an equivalence
relation, equal keys must hash equally, and hashes must be stable while a key
is stored.

See `docs/spec.md` for the authoritative behavior contract.
