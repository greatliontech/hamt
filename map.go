package hamt

import "math/bits"

const (
	fragmentBits = 5
	fragmentMask = 1<<fragmentBits - 1
)

// Map is an immutable hash map.
type Map[K, V any] struct {
	root   *node[K, V]
	size   int
	hasher Hasher[K]
}

// NewMap returns an empty immutable map.
func NewMap[K, V any](hasher Hasher[K]) Map[K, V] {
	return Map[K, V]{hasher: hasher}
}

// Len returns the number of key/value pairs in the map.
func (m Map[K, V]) Len() int {
	return m.size
}

// Get returns the value for key and whether key exists.
func (m Map[K, V]) Get(key K) (V, bool) {
	var zero V
	if m.root == nil {
		return zero, false
	}
	m.mustHasher()
	return m.root.get(key, m.hasher.Hash(key), 0, m.hasher)
}

// Set returns a map with key associated with value.
func (m Map[K, V]) Set(key K, value V) Map[K, V] {
	m.mustHasher()
	e := entry[K, V]{key: key, value: value, hash: m.hasher.Hash(key)}
	if m.root == nil {
		return Map[K, V]{root: newSingleNode(e, 0), size: 1, hasher: m.hasher}
	}

	root, added := m.root.set(e, 0, m.hasher)
	if added {
		return Map[K, V]{root: root, size: m.size + 1, hasher: m.hasher}
	}
	return Map[K, V]{root: root, size: m.size, hasher: m.hasher}
}

// Delete returns a map with key removed.
func (m Map[K, V]) Delete(key K) Map[K, V] {
	if m.root == nil {
		return m
	}
	m.mustHasher()
	root, removed := m.root.delete(key, m.hasher.Hash(key), 0, m.hasher)
	if !removed {
		return m
	}
	return Map[K, V]{root: root, size: m.size - 1, hasher: m.hasher}
}

// Range calls fn for each key/value pair until fn returns false or iteration
// completes.
func (m Map[K, V]) Range(fn func(K, V) bool) {
	if m.root == nil {
		return
	}
	m.root.each(fn)
}

func (m Map[K, V]) mustHasher() {
	if m.hasher == nil {
		panic("hamt: nil Hasher")
	}
}

type entry[K, V any] struct {
	key   K
	value V
	hash  uint64
}

type collisionEntry[K, V any] struct {
	key   K
	value V
}

// node is a branch when collision is nil and a collision node otherwise.
// A collision node uses none of the branch fields; the collision payload
// lives behind a pointer so branch nodes, which dominate any map with a
// reasonable hasher, do not carry storage for it.
type node[K, V any] struct {
	dataMap   uint32
	nodeMap   uint32
	entries   []entry[K, V]
	children  []*node[K, V]
	collision *collisionNode[K, V]
}

// collisionNode holds entries whose keys are distinct but whose full 64-bit
// hashes are identical.
type collisionNode[K, V any] struct {
	hash    uint64
	entries []collisionEntry[K, V]
}

func newSingleNode[K, V any](e entry[K, V], shift uint) *node[K, V] {
	bit := bitpos(fragment(e.hash, shift))
	return &node[K, V]{dataMap: bit, entries: []entry[K, V]{e}}
}

func newCollisionNode[K, V any](hash uint64, entries []entry[K, V]) *node[K, V] {
	collisions := make([]collisionEntry[K, V], len(entries))
	for i, e := range entries {
		collisions[i] = collisionEntry[K, V]{key: e.key, value: e.value}
	}
	return &node[K, V]{collision: &collisionNode[K, V]{hash: hash, entries: collisions}}
}

func (n *node[K, V]) isCollision() bool {
	return n.collision != nil
}

func (n *node[K, V]) get(key K, hash uint64, shift uint, h Hasher[K]) (V, bool) {
	var zero V
	for {
		if c := n.collision; c != nil {
			if hash != c.hash {
				return zero, false
			}
			for _, e := range c.entries {
				if h.Equal(e.key, key) {
					return e.value, true
				}
			}
			return zero, false
		}

		bit := bitpos(fragment(hash, shift))
		if n.dataMap&bit != 0 {
			e := n.entries[index(n.dataMap, bit)]
			if e.hash == hash && h.Equal(e.key, key) {
				return e.value, true
			}
			return zero, false
		}
		if n.nodeMap&bit == 0 {
			return zero, false
		}
		n = n.children[index(n.nodeMap, bit)]
		shift += fragmentBits
	}
}

func (n *node[K, V]) set(e entry[K, V], shift uint, h Hasher[K]) (*node[K, V], bool) {
	if n.isCollision() {
		return n.setCollision(e, shift, h)
	}

	bit := bitpos(fragment(e.hash, shift))
	if n.dataMap&bit != 0 {
		idx := index(n.dataMap, bit)
		old := n.entries[idx]
		if old.hash == e.hash && h.Equal(old.key, e.key) {
			return n.cloneWithEntry(idx, e), false
		}

		child := mergeEntries(old, e, shift+fragmentBits)
		return n.cloneWithEntryReplacedByChild(bit, idx, child), true
	}

	if n.nodeMap&bit != 0 {
		idx := index(n.nodeMap, bit)
		child, added := n.children[idx].set(e, shift+fragmentBits, h)
		return n.cloneWithChild(idx, child), added
	}

	return n.cloneWithInsertedEntry(bit, e), true
}

func (n *node[K, V]) setCollision(e entry[K, V], shift uint, h Hasher[K]) (*node[K, V], bool) {
	if e.hash != n.collision.hash {
		return pushCollision(n, e, shift), true
	}

	for i, old := range n.collision.entries {
		if h.Equal(old.key, e.key) {
			return n.cloneCollisionWithEntry(i, e), false
		}
	}

	return n.cloneCollisionWithInsertedEntry(e), true
}

// pushCollision resolves an insert whose hash differs from the collision
// node's hash. The collision node moves down unchanged — shared, since
// nothing in it changes — along the fragments the two hashes still agree
// on, and e lands where they diverge. The hashes differ in some bit below
// 64, so a diverging fragment exists at shift <= 60 and the recursion
// terminates.
func pushCollision[K, V any](collision *node[K, V], e entry[K, V], shift uint) *node[K, V] {
	collisionBit := bitpos(fragment(collision.collision.hash, shift))
	entryBit := bitpos(fragment(e.hash, shift))
	if collisionBit != entryBit {
		return &node[K, V]{
			dataMap:  entryBit,
			nodeMap:  collisionBit,
			entries:  []entry[K, V]{e},
			children: []*node[K, V]{collision},
		}
	}
	child := pushCollision(collision, e, shift+fragmentBits)
	return &node[K, V]{nodeMap: collisionBit, children: []*node[K, V]{child}}
}

func (n *node[K, V]) delete(key K, hash uint64, shift uint, h Hasher[K]) (*node[K, V], bool) {
	if n.isCollision() {
		return n.deleteCollision(key, hash, h)
	}

	bit := bitpos(fragment(hash, shift))
	if n.dataMap&bit != 0 {
		idx := index(n.dataMap, bit)
		e := n.entries[idx]
		if e.hash != hash || !h.Equal(e.key, key) {
			return n, false
		}
		clone := n.cloneWithoutEntrySharedChildren(bit, idx)
		if clone.isEmpty() {
			return nil, true
		}
		return clone, true
	}

	if n.nodeMap&bit == 0 {
		return n, false
	}

	idx := index(n.nodeMap, bit)
	child, removed := n.children[idx].delete(key, hash, shift+fragmentBits, h)
	if !removed {
		return n, false
	}

	if child != nil {
		if e, ok := child.singleton(); ok {
			return n.cloneWithChildReplacedByEntry(bit, idx, e), true
		}
		return n.cloneWithChild(idx, child), true
	}
	panic("hamt: delete produced empty child")
}

func (n *node[K, V]) deleteCollision(key K, hash uint64, h Hasher[K]) (*node[K, V], bool) {
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
		return n.cloneCollisionWithoutEntry(i), true
	}
	return n, false
}

func (n *node[K, V]) each(fn func(K, V) bool) bool {
	if c := n.collision; c != nil {
		for _, e := range c.entries {
			if !fn(e.key, e.value) {
				return false
			}
		}
		return true
	}

	// dataMap and nodeMap are disjoint, so each set bit of the union is
	// exactly one entry or one child, and ascending bit order visits the
	// slices sequentially.
	entryIdx := 0
	childIdx := 0
	for bitmap := n.dataMap | n.nodeMap; bitmap != 0; bitmap &= bitmap - 1 {
		bit := bitmap & -bitmap
		if n.dataMap&bit != 0 {
			e := n.entries[entryIdx]
			entryIdx++
			if !fn(e.key, e.value) {
				return false
			}
		} else {
			child := n.children[childIdx]
			childIdx++
			if !child.each(fn) {
				return false
			}
		}
	}
	return true
}

func (n *node[K, V]) singleton() (entry[K, V], bool) {
	var zero entry[K, V]
	if c := n.collision; c != nil {
		if len(c.entries) == 1 {
			e := c.entries[0]
			return entry[K, V]{key: e.key, value: e.value, hash: c.hash}, true
		}
		return zero, false
	}
	if len(n.entries) == 1 && len(n.children) == 0 {
		return n.entries[0], true
	}
	return zero, false
}

func (n *node[K, V]) cloneCollisionWithEntry(idx int, e entry[K, V]) *node[K, V] {
	entries := append([]collisionEntry[K, V](nil), n.collision.entries...)
	entries[idx] = collisionEntry[K, V]{key: e.key, value: e.value}
	return &node[K, V]{collision: &collisionNode[K, V]{hash: n.collision.hash, entries: entries}}
}

func (n *node[K, V]) cloneCollisionWithInsertedEntry(e entry[K, V]) *node[K, V] {
	old := n.collision.entries
	entries := make([]collisionEntry[K, V], len(old)+1)
	copy(entries, old)
	entries[len(old)] = collisionEntry[K, V]{key: e.key, value: e.value}
	return &node[K, V]{collision: &collisionNode[K, V]{hash: n.collision.hash, entries: entries}}
}

func (n *node[K, V]) cloneCollisionWithoutEntry(idx int) *node[K, V] {
	return &node[K, V]{collision: &collisionNode[K, V]{hash: n.collision.hash, entries: removeCollision(n.collision.entries, idx)}}
}

func (n *node[K, V]) cloneWithEntry(idx int, e entry[K, V]) *node[K, V] {
	clone := *n
	clone.entries = append([]entry[K, V](nil), n.entries...)
	clone.entries[idx] = e
	return &clone
}

func (n *node[K, V]) cloneWithInsertedEntry(bit uint32, e entry[K, V]) *node[K, V] {
	clone := *n
	clone.dataMap |= bit
	clone.entries = insertEntryCopy(n.entries, index(n.dataMap, bit), e)
	return &clone
}

func (n *node[K, V]) cloneWithEntryReplacedByChild(bit uint32, entryIdx int, child *node[K, V]) *node[K, V] {
	clone := *n
	clone.dataMap &^= bit
	clone.entries = removeEntry(n.entries, entryIdx)
	clone.nodeMap |= bit
	clone.children = insertChildCopy(n.children, index(n.nodeMap, bit), child)
	return &clone
}

func (n *node[K, V]) cloneWithoutEntrySharedChildren(bit uint32, idx int) *node[K, V] {
	clone := *n
	clone.dataMap &^= bit
	clone.entries = removeEntry(n.entries, idx)
	return &clone
}

func (n *node[K, V]) cloneWithChild(idx int, child *node[K, V]) *node[K, V] {
	clone := *n
	clone.children = append([]*node[K, V](nil), n.children...)
	clone.children[idx] = child
	return &clone
}

func (n *node[K, V]) cloneWithChildReplacedByEntry(bit uint32, childIdx int, e entry[K, V]) *node[K, V] {
	clone := *n
	clone.nodeMap &^= bit
	clone.children = removeChild(n.children, childIdx)
	clone.dataMap |= bit
	clone.entries = insertEntryCopy(n.entries, index(n.dataMap, bit), e)
	return &clone
}

func (n *node[K, V]) isEmpty() bool {
	return n.collision == nil && n.dataMap == 0 && n.nodeMap == 0
}

// mergeEntries builds the subtree holding two entries that share the path
// down to shift. Equal hashes collide; unequal hashes diverge at some
// fragment at shift <= 60, terminating the recursion.
func mergeEntries[K, V any](a, b entry[K, V], shift uint) *node[K, V] {
	if a.hash == b.hash {
		return newCollisionNode(a.hash, []entry[K, V]{a, b})
	}

	aBit := bitpos(fragment(a.hash, shift))
	bBit := bitpos(fragment(b.hash, shift))
	if aBit != bBit {
		n := &node[K, V]{dataMap: aBit | bBit}
		if aBit < bBit {
			n.entries = []entry[K, V]{a, b}
		} else {
			n.entries = []entry[K, V]{b, a}
		}
		return n
	}

	child := mergeEntries(a, b, shift+fragmentBits)
	return &node[K, V]{nodeMap: aBit, children: []*node[K, V]{child}}
}

func fragment(hash uint64, shift uint) uint32 {
	return uint32(hash>>shift) & fragmentMask
}

func bitpos(fragment uint32) uint32 {
	return 1 << fragment
}

func index(bitmap, bit uint32) int {
	return bits.OnesCount32(bitmap & (bit - 1))
}

func insertEntryCopy[K, V any](entries []entry[K, V], idx int, e entry[K, V]) []entry[K, V] {
	out := make([]entry[K, V], len(entries)+1)
	copy(out, entries[:idx])
	out[idx] = e
	copy(out[idx+1:], entries[idx:])
	return out
}

func removeEntry[K, V any](entries []entry[K, V], idx int) []entry[K, V] {
	out := make([]entry[K, V], len(entries)-1)
	copy(out, entries[:idx])
	copy(out[idx:], entries[idx+1:])
	return out
}

func removeCollision[K, V any](entries []collisionEntry[K, V], idx int) []collisionEntry[K, V] {
	out := make([]collisionEntry[K, V], len(entries)-1)
	copy(out, entries[:idx])
	copy(out[idx:], entries[idx+1:])
	return out
}

func insertChildCopy[K, V any](children []*node[K, V], idx int, child *node[K, V]) []*node[K, V] {
	out := make([]*node[K, V], len(children)+1)
	copy(out, children[:idx])
	out[idx] = child
	copy(out[idx+1:], children[idx:])
	return out
}

func removeChild[K, V any](children []*node[K, V], idx int) []*node[K, V] {
	out := make([]*node[K, V], len(children)-1)
	copy(out, children[:idx])
	copy(out[idx:], children[idx+1:])
	return out
}
