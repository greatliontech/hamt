// Package hamt provides a generic immutable hash map.
//
// Maps are persistent: Set and Delete return new maps while previous maps
// remain valid snapshots. New keys a map by language equality (==);
// NewWithHasher keys it through a caller-provided Hasher for custom key
// identity or non-comparable key types.
//
// For bulk construction, use Builder. A builder mutates private state until Map
// returns the built immutable map and invalidates the builder.
package hamt
