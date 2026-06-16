# hamt

`hamt` is a generic immutable hash map for Go.

The map is persistent: `Set` and `Delete` return new maps while previous maps remain valid snapshots. Keys are compared and hashed through a caller-provided `Hasher[K]`.

```go
m := hamt.NewMap[string, int](hamt.StringHasher{})
m = m.Set("jane", 100)

v, ok := m.Get("jane")
fmt.Println(v, ok) // 100 true
```

For bulk construction, use a builder:

```go
b := hamt.NewBuilder[string, int](hamt.StringHasher{})
b.Set("jane", 100)
b.Set("susy", 200)
m := b.Map()
```

See `docs/spec.md` for the authoritative behavior contract.
