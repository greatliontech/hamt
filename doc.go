// Package hamt provides a generic immutable hash map.
//
// Maps are persistent: Set and Delete return new maps while previous maps
// remain valid snapshots. Keys are hashed and compared through a caller-provided
// Hasher.
//
// For bulk construction, use Builder. A builder mutates private state until Map
// returns the built immutable map and invalidates the builder.
package hamt
