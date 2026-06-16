package hamt

import "math/bits"

const (
	fragmentBits = 5
	fragmentMask = 1<<fragmentBits - 1
)

// Hasher hashes keys and checks them for equality.
type Hasher[K any] interface {
	Hash(K) uint64
	Equal(K, K) bool
}

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

type node[K, V any] struct {
	dataMap  uint32
	nodeMap  uint32
	entries  []entry[K, V]
	children []*node[K, V]

	collision     bool
	collisionHash uint64
	collisions    []entry[K, V]
}

func newSingleNode[K, V any](e entry[K, V], shift uint) *node[K, V] {
	bit := bitpos(fragment(e.hash, shift))
	return &node[K, V]{dataMap: bit, entries: []entry[K, V]{e}}
}

func newCollisionNode[K, V any](hash uint64, entries []entry[K, V]) *node[K, V] {
	collisions := make([]entry[K, V], len(entries))
	copy(collisions, entries)
	return &node[K, V]{collision: true, collisionHash: hash, collisions: collisions}
}

func (n *node[K, V]) get(key K, hash uint64, shift uint, h Hasher[K]) (V, bool) {
	var zero V
	if n.collision {
		if hash != n.collisionHash {
			return zero, false
		}
		for _, e := range n.collisions {
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
	if n.nodeMap&bit != 0 {
		return n.children[index(n.nodeMap, bit)].get(key, hash, shift+fragmentBits, h)
	}
	return zero, false
}

func (n *node[K, V]) set(e entry[K, V], shift uint, h Hasher[K]) (*node[K, V], bool) {
	if n.collision {
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
	if e.hash != n.collisionHash {
		var root *node[K, V]
		for _, existing := range n.collisions {
			root = insertKnown(root, existing, shift, h)
		}
		root = insertKnown(root, e, shift, h)
		return root, true
	}

	for i, old := range n.collisions {
		if h.Equal(old.key, e.key) {
			clone := n.clone()
			clone.collisions[i] = e
			return clone, false
		}
	}

	clone := n.clone()
	clone.collisions = append(clone.collisions, e)
	return clone, true
}

func (n *node[K, V]) delete(key K, hash uint64, shift uint, h Hasher[K]) (*node[K, V], bool) {
	if n.collision {
		return n.deleteCollision(key, hash, shift, h)
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

	clone := n.cloneWithoutChildSharedEntries(bit, idx)
	if clone.isEmpty() {
		return nil, true
	}
	return clone, true
}

func (n *node[K, V]) deleteCollision(key K, hash uint64, shift uint, h Hasher[K]) (*node[K, V], bool) {
	if hash != n.collisionHash {
		return n, false
	}
	for i, e := range n.collisions {
		if !h.Equal(e.key, key) {
			continue
		}
		if len(n.collisions) == 1 {
			return nil, true
		}
		clone := n.clone()
		clone.collisions = removeEntry(clone.collisions, i)
		return clone, true
	}
	return n, false
}

func (n *node[K, V]) each(fn func(K, V) bool) bool {
	if n.collision {
		for _, e := range n.collisions {
			if !fn(e.key, e.value) {
				return false
			}
		}
		return true
	}

	entryIdx := 0
	childIdx := 0
	for bit := uint32(1); bit != 0; bit <<= 1 {
		if n.dataMap&bit != 0 {
			e := n.entries[entryIdx]
			entryIdx++
			if !fn(e.key, e.value) {
				return false
			}
		}
		if n.nodeMap&bit != 0 {
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
	if n.collision {
		if len(n.collisions) == 1 {
			return n.collisions[0], true
		}
		return zero, false
	}
	if len(n.entries) == 1 && len(n.children) == 0 {
		return n.entries[0], true
	}
	return zero, false
}

func (n *node[K, V]) clone() *node[K, V] {
	clone := *n
	if n.entries != nil {
		clone.entries = append([]entry[K, V](nil), n.entries...)
	}
	if n.children != nil {
		clone.children = append([]*node[K, V](nil), n.children...)
	}
	if n.collisions != nil {
		clone.collisions = append([]entry[K, V](nil), n.collisions...)
	}
	return &clone
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

func (n *node[K, V]) cloneWithoutChildSharedEntries(bit uint32, idx int) *node[K, V] {
	clone := *n
	clone.nodeMap &^= bit
	clone.children = removeChild(n.children, idx)
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
	return !n.collision && n.dataMap == 0 && n.nodeMap == 0
}

func mergeEntries[K, V any](a, b entry[K, V], shift uint) *node[K, V] {
	if shift >= 64 || a.hash == b.hash {
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

func insertKnown[K, V any](root *node[K, V], e entry[K, V], shift uint, h Hasher[K]) *node[K, V] {
	if root == nil {
		return newSingleNode(e, shift)
	}
	root, _ = root.set(e, shift, h)
	return root
}

func fragment(hash uint64, shift uint) uint32 {
	if shift >= 64 {
		return 0
	}
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

// IntHasher hashes int keys.
type IntHasher struct{}

func (IntHasher) Hash(v int) uint64   { return mix64(uint64(v)) }
func (IntHasher) Equal(a, b int) bool { return a == b }

// Uint64Hasher hashes uint64 keys.
type Uint64Hasher struct{}

func (Uint64Hasher) Hash(v uint64) uint64   { return mix64(v) }
func (Uint64Hasher) Equal(a, b uint64) bool { return a == b }

// StringHasher hashes string keys.
type StringHasher struct{}

func (StringHasher) Hash(s string) uint64 {
	const prime = 1099511628211
	var hash uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= prime
	}
	return hash
}

func (StringHasher) Equal(a, b string) bool { return a == b }

func mix64(v uint64) uint64 {
	v += 0x9e3779b97f4a7c15
	v = (v ^ (v >> 30)) * 0xbf58476d1ce4e5b9
	v = (v ^ (v >> 27)) * 0x94d049bb133111eb
	return v ^ (v >> 31)
}
