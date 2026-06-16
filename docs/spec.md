# Immutable Map Specification

This package provides a generic immutable hash map for Go.

## Public Contract

- `Map[K, V]` stores at most one value for each key according to the map's `Hasher[K]`.
- `NewMap[K, V](hasher)` creates an empty map. `hasher` must be non-nil before an operation needs to hash a key.
- `Len` returns the number of reachable key/value pairs.
- `Get` returns the value for a key and whether the key exists.
- `Set` returns a map where the key is associated with the supplied value.
- `Delete` returns a map where the key is absent. Deleting a missing key returns an equivalent map.
- `Range` visits each key/value pair until the callback returns false or all entries have been visited.
- `Builder[K, V]` efficiently constructs a map through mutable `Set` and `Delete` operations before returning an immutable `Map`.

## Immutability

Every update operation is persistent. A map returned by `Set` or `Delete` may share internal structure with its source map, but it must not mutate any previously returned map. A map is safe for concurrent readers after publication when callers do not mutate data reachable through keys or values.

## Builder

`NewBuilder[K, V](hasher)` creates an empty builder. `Builder.Set` and `Builder.Delete` mutate the builder and do not return a new builder. `Builder.Map` returns the built immutable map and invalidates the builder. Any later builder method call after `Map` must panic.

The map returned by `Builder.Map` follows the same immutability, hashing, equality, collision, iteration, and structural canonicalization rules as any other `Map`.

## Hashing And Equality

`Hasher[K]` defines the key identity contract:

- `Equal(a, b)` is true exactly when `a` and `b` are the same map key.
- If `Equal(a, b)` is true, `Hash(a)` and `Hash(b)` must be equal.
- If the same key is hashed multiple times while it is stored in a map, the hash must be stable.

Violating this contract makes lookups, updates, and deletes undefined for the affected keys.

## Collision Semantics

Distinct keys may have the same hash. The map must preserve all distinct colliding keys and use `Equal` to disambiguate them.

## Iteration

Iteration order is deterministic for a fixed map, hash function, and insertion history, but it is not sorted and is not part of the compatibility contract.

## Structural Canonicalization

Deletes must remove empty branches. When a delete leaves an intermediate branch containing only a single direct entry, that branch is collapsed so future operations do not retain avoidable singleton path nodes.

## Benchmark Baselines

Benchmark comparisons target:

- `github.com/benbjohnson/immutable` for a generic immutable map with an external hasher.
