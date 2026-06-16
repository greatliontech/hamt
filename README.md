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

## Development

This repository uses [Task](https://taskfile.dev/) for common command sets:

- `task test`: run unit tests.
- `task test:race`: run tests with race detection.
- `task check`: run standard verification.
- `task bench:quick`: run a one-iteration benchmark smoke test.
- `task bench`: run the benchmark matrix. Override with `BENCH_TIME=5s BENCH_COUNT=10 task bench`.
- `task bench:builder`: run builder benchmarks only.
- `task bench:profile`: write CPU and allocation profiles under `/tmp/opencode/hamt-profiles` by default.
- `task pprof:cpu`: show the CPU profile top output.
- `task pprof:mem`: show the allocation profile top output.
