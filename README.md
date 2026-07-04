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

## Comparison with benbjohnson/immutable

Median ns/op against [`benbjohnson/immutable`](https://github.com/benbjohnson/immutable)
v0.4.3 `Map`, the library `hamt` was built to replace. Both sides use the
same splitmix64-derived hasher over `uint64` keys (`immutable` hashes to 32
bits, `hamt` to 64 — each library's own design) and run in one binary so
samples interleave. benchstat over 8 runs pinned to the P-cores of an Intel
Core Ultra 7 258V, Go 1.26; every delta below is p < 0.01, sizes are
entries in the map.

| Operation    | Library   |     1 |     8 |    32 |    1024 |
| ------------ | --------- | ----: | ----: | ----: | ------: |
| Get (hit)    | immutable |   4.5 |   7.9 |  11.5 |    23.0 |
|              | hamt      |   4.9 |   4.8 |   8.7 |    15.7 |
| Set (insert) | immutable |  78.6 | 799.0 | 181.6 |   342.6 |
|              | hamt      |  55.1 |  86.7 | 116.8 |   213.8 |
| Set (update) | immutable |  70.2 | 141.8 | 285.6 |   461.2 |
|              | hamt      |  52.6 |  94.5 | 140.7 |   302.6 |
| Delete (hit) | immutable |  44.9 | 118.8 | 248.9 |   501.9 |
|              | hamt      |  47.0 |  88.5 | 158.3 |   374.2 |
| Build (Set)  | immutable |  83.6 | 975.9 | 8,075 | 439,800 |
|              | hamt      |  67.6 | 587.3 | 4,158 | 301,200 |
| Builder      | immutable |  64.0 | 354.2 | 3,160 |  91,330 |
|              | hamt      |  53.2 | 217.5 | 1,614 |  50,810 |

Suite geomean: `hamt` is 37% faster, allocating 34% fewer bytes and 44%
fewer objects per operation. `immutable` wins two cells: Get at a single
entry (by 0.4 ns) and Delete at a single entry allocates 49 B to `hamt`'s
64 B. The 8-entry insert outlier is `immutable` promoting its compact
array node to a full HAMT node; `hamt` has no such cliff.
