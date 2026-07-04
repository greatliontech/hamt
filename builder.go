package hamt

// Builder efficiently constructs an immutable map.
type Builder[K, V any] struct {
	state *builderState[K, V]
}

type builderState[K, V any] struct {
	root   *node[K, V]
	size   int
	hasher Hasher[K]
	valid  bool
}

// NewBuilder returns an empty builder.
func NewBuilder[K, V any](hasher Hasher[K]) *Builder[K, V] {
	return &Builder[K, V]{state: &builderState[K, V]{hasher: hasher, valid: true}}
}

// Len returns the number of key/value pairs in the builder.
func (b *Builder[K, V]) Len() int {
	state := b.mustState()
	return state.size
}

// Get returns the value for key and whether key exists.
func (b *Builder[K, V]) Get(key K) (V, bool) {
	state := b.mustState()
	var zero V
	if state.root == nil {
		return zero, false
	}
	state.mustHasher()
	return state.root.get(key, state.hasher.Hash(key), 0, state.hasher)
}

// Set associates key with value in the builder.
func (b *Builder[K, V]) Set(key K, value V) {
	state := b.mustState()
	state.mustHasher()
	e := entry[K, V]{key: key, value: value, hash: state.hasher.Hash(key)}
	if state.root == nil {
		state.root = newSingleNode(e, 0)
		state.size = 1
		return
	}

	var added bool
	state.root, added = state.root.setMutable(e, 0, state.hasher)
	if added {
		state.size++
	}
}

// Delete removes key from the builder.
func (b *Builder[K, V]) Delete(key K) {
	state := b.mustState()
	if state.root == nil {
		return
	}
	state.mustHasher()

	var removed bool
	state.root, removed = state.root.deleteMutable(key, state.hasher.Hash(key), 0, state.hasher)
	if removed {
		state.size--
	}
}

// Map returns the built map and invalidates the builder.
func (b *Builder[K, V]) Map() Map[K, V] {
	state := b.mustState()
	m := Map[K, V]{root: state.root, size: state.size, hasher: state.hasher}
	state.root = nil
	state.size = 0
	state.hasher = nil
	state.valid = false
	b.state = nil
	return m
}

func (b *Builder[K, V]) mustState() *builderState[K, V] {
	if b == nil || b.state == nil || !b.state.valid {
		panic("hamt: builder invalid after Map")
	}
	return b.state
}

func (s *builderState[K, V]) mustHasher() {
	if s.hasher == nil {
		panic("hamt: nil Hasher")
	}
}

// The node methods below are the builder's in-place counterparts of the
// persistent operations in map.go. They may mutate the receiver and any
// node beneath it, which is sound only because every node reachable from a
// builder was created by that builder and Map() invalidates it before the
// tree is published.

func (n *node[K, V]) setMutable(e entry[K, V], shift uint, h Hasher[K]) (*node[K, V], bool) {
	if n.isCollision() {
		return n.setCollisionMutable(e, shift, h)
	}

	bit := bitpos(fragment(e.hash, shift))
	if n.dataMap&bit != 0 {
		idx := index(n.dataMap, bit)
		old := n.entries[idx]
		if old.hash == e.hash && h.Equal(old.key, e.key) {
			n.entries[idx] = e
			return n, false
		}

		child := mergeEntries(old, e, shift+fragmentBits)
		n.dataMap &^= bit
		n.entries = removeEntryMutable(n.entries, idx)
		childIdx := index(n.nodeMap, bit)
		n.nodeMap |= bit
		n.children = insertChildMutable(n.children, childIdx, child)
		return n, true
	}

	if n.nodeMap&bit != 0 {
		idx := index(n.nodeMap, bit)
		child, added := n.children[idx].setMutable(e, shift+fragmentBits, h)
		n.children[idx] = child
		return n, added
	}

	idx := index(n.dataMap, bit)
	n.dataMap |= bit
	n.entries = insertEntryMutable(n.entries, idx, e)
	return n, true
}

func (n *node[K, V]) setCollisionMutable(e entry[K, V], shift uint, h Hasher[K]) (*node[K, V], bool) {
	c := n.collision
	if e.hash != c.hash {
		return pushCollision(n, e, shift), true
	}

	for i, old := range c.entries {
		if h.Equal(old.key, e.key) {
			c.entries[i] = collisionEntry[K, V]{key: e.key, value: e.value}
			return n, false
		}
	}

	c.entries = append(c.entries, collisionEntry[K, V]{key: e.key, value: e.value})
	return n, true
}

func (n *node[K, V]) deleteMutable(key K, hash uint64, shift uint, h Hasher[K]) (*node[K, V], bool) {
	if n.isCollision() {
		return n.deleteCollisionMutable(key, hash, h)
	}

	bit := bitpos(fragment(hash, shift))
	if n.dataMap&bit != 0 {
		idx := index(n.dataMap, bit)
		e := n.entries[idx]
		if e.hash != hash || !h.Equal(e.key, key) {
			return n, false
		}
		n.dataMap &^= bit
		n.entries = removeEntryMutable(n.entries, idx)
		if n.isEmpty() {
			return nil, true
		}
		return n, true
	}

	if n.nodeMap&bit == 0 {
		return n, false
	}

	idx := index(n.nodeMap, bit)
	child, removed := n.children[idx].deleteMutable(key, hash, shift+fragmentBits, h)
	if !removed {
		return n, false
	}

	if child != nil {
		if e, ok := child.singleton(); ok {
			n.nodeMap &^= bit
			n.children = removeChildMutable(n.children, idx)
			entryIdx := index(n.dataMap, bit)
			n.dataMap |= bit
			n.entries = insertEntryMutable(n.entries, entryIdx, e)
			return n, true
		}
		n.children[idx] = child
		return n, true
	}
	panic("hamt: delete produced empty child")
}

func (n *node[K, V]) deleteCollisionMutable(key K, hash uint64, h Hasher[K]) (*node[K, V], bool) {
	c := n.collision
	if hash != c.hash {
		return n, false
	}
	for i, e := range c.entries {
		if !h.Equal(e.key, key) {
			continue
		}
		if len(c.entries) == 1 {
			return nil, true
		}
		c.entries = removeCollisionMutable(c.entries, i)
		return n, true
	}
	return n, false
}

func insertEntryMutable[K, V any](entries []entry[K, V], idx int, e entry[K, V]) []entry[K, V] {
	entries = append(entries, entry[K, V]{})
	copy(entries[idx+1:], entries[idx:])
	entries[idx] = e
	return entries
}

func removeEntryMutable[K, V any](entries []entry[K, V], idx int) []entry[K, V] {
	copy(entries[idx:], entries[idx+1:])
	var zero entry[K, V]
	entries[len(entries)-1] = zero
	return entries[:len(entries)-1]
}

func removeCollisionMutable[K, V any](entries []collisionEntry[K, V], idx int) []collisionEntry[K, V] {
	copy(entries[idx:], entries[idx+1:])
	var zero collisionEntry[K, V]
	entries[len(entries)-1] = zero
	return entries[:len(entries)-1]
}

func insertChildMutable[K, V any](children []*node[K, V], idx int, child *node[K, V]) []*node[K, V] {
	children = append(children, nil)
	copy(children[idx+1:], children[idx:])
	children[idx] = child
	return children
}

func removeChildMutable[K, V any](children []*node[K, V], idx int) []*node[K, V] {
	copy(children[idx:], children[idx+1:])
	children[len(children)-1] = nil
	return children[:len(children)-1]
}
