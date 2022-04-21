package iavl

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	dbm "github.com/tendermint/tendermint/libs/db"
)

// ImmutableTree is a container for an immutable AVL+ ImmutableTree. Changes are performed by
// swapping the internal root with a new one, while the container is mutable.
// Note that this tree is not thread-safe.
type ImmutableTree struct {
	root          *Node
	lastSavedRoot *Node // The most recently saved root node
	ndb           *nodeDB
	version       int64
	mtx           sync.Mutex // used when get root from db
	nodeVersions  *NodeVersions
	isNotEmpty    bool // so the tree is empty by default
}

// NewImmutableTree creates both in-memory and persistent instances
func NewImmutableTree(db dbm.DB, cacheSize int) *ImmutableTree {
	if db == nil {
		// In-memory Tree.
		return &ImmutableTree{}
	}
	return &ImmutableTree{
		// NodeDB-backed Tree.
		ndb:          NewNodeDB(db, cacheSize),
		nodeVersions: NewNodeVersions(defaultMaxVersions, defaultMaxNodes, 0),
	}
}

func GetRoot(t *ImmutableTree) *Node {
	return t.getRoot()
}

func (t *ImmutableTree) getRoot() *Node {
	if t.root != nil {
		// this handles most cases.
		return t.root
	}
	// when t.root is nil, either the root is also pruned from memory,
	// or the root node is removed and tree is empty.

	if !t.isNotEmpty {
		// root node is deleted, this can happen between two SaveVersion
		return nil
	}

	// root node is pruned, this can happen at the first time of getting root after last SaveVersion
	t.mtx.Lock()
	t.root = t.lastSavedRoot // we can ensure lastSaveRoot is not nil when the tree is not empty
	t.root.loadVersion = t.version
	t.nodeVersions.Inc1(t.root.loadVersion)
	t.mtx.Unlock()
	return t.root
}

func (t *ImmutableTree) updateLastSaveRoot() {
	root := t.root
	if root == nil {
		t.lastSavedRoot = nil
		return
	}
	// only keep the root node itself without the left and right node.
	t.lastSavedRoot = &Node{
		key:         root.key,
		value:       root.value,
		height:      root.height,
		version:     root.version,
		size:        root.size,
		hash:        root.hash,
		leftHash:    root.leftHash,
		leftNode:    nil,
		rightHash:   root.rightHash,
		rightNode:   nil,
		persisted:   root.persisted,
		loadVersion: t.version,
	}
}

// String returns a string representation of Tree.
func (t *ImmutableTree) String() string {
	leaves := []string{}
	t.Iterate(func(key []byte, val []byte) (stop bool) {
		leaves = append(leaves, fmt.Sprintf("%x: %x", key, val))
		return false
	})
	return "Tree{" + strings.Join(leaves, ", ") + "}"
}

// Size returns the number of leaf nodes in the tree.
func (t *ImmutableTree) Size() int64 {
	root := t.getRoot()
	if root == nil {
		return 0
	}
	return root.size
}

// Version returns the version of the tree.
func (t *ImmutableTree) Version() int64 {
	return t.version
}

// Height returns the height of the tree.
func (t *ImmutableTree) Height() int8 {
	root := t.getRoot()
	if root == nil {
		return 0
	}
	return root.height
}

// Has returns whether or not a key exists.
func (t *ImmutableTree) Has(key []byte) bool {
	root := t.getRoot()
	if root == nil {
		return false
	}
	return root.has(t, key)
}

// Hash returns the root hash.
func (t *ImmutableTree) Hash() []byte {
	root := t.getRoot()
	if root == nil {
		return nil
	}
	hash, _ := root.hashWithCount()
	return hash
}

// hashWithCount returns the root hash and hash count.
func (t *ImmutableTree) hashWithCount() ([]byte, int64) {
	root := t.getRoot()
	if root == nil {
		return nil, 0
	}
	return root.hashWithCount()
}

// Get returns the index and value of the specified key if it exists, or nil
// and the next index, if it doesn't.
func (t *ImmutableTree) Get(key []byte) (index int64, value []byte) {
	root := t.getRoot()
	if root == nil {
		return 0, nil
	}
	return root.get(t, key)
}

// GetByIndex gets the key and value at the specified index.
func (t *ImmutableTree) GetByIndex(index int64) (key []byte, value []byte) {
	root := t.getRoot()
	if root == nil {
		return nil, nil
	}
	return root.getByIndex(t, index)
}

// Iterate iterates over all keys of the tree, in order.
func (t *ImmutableTree) Iterate(fn func(key []byte, value []byte) bool) (stopped bool) {
	root := t.getRoot()
	if root == nil {
		return false
	}
	return root.traverse(t, true, func(node *Node) bool {
		if node.height == 0 {
			return fn(node.key, node.value)
		}
		return false
	})
}

// used by state syncing
func (t *ImmutableTree) IterateFirst(fn func(nodeBytes []byte)) {
	root := t.getRoot()
	if root == nil {
		return
	}
	root.traverseFirst(t, true, func(node *Node) bool {
		var b bytes.Buffer
		if err := node.writeBytes(&b); err != nil {
			panic(err)
		}
		fn(b.Bytes())
		return false
	})
}

// IterateRange makes a callback for all nodes with key between start and end non-inclusive.
// If either are nil, then it is open on that side (nil, nil is the same as Iterate)
func (t *ImmutableTree) IterateRange(start, end []byte, ascending bool, fn func(key []byte, value []byte) bool) (stopped bool) {
	root := t.getRoot()
	if root == nil {
		return false
	}
	return root.traverseInRange(t, start, end, ascending, false, 0, func(node *Node, _ uint8) bool {
		if node.height == 0 {
			return fn(node.key, node.value)
		}
		return false
	})
}

// IterateRangeInclusive makes a callback for all nodes with key between start and end inclusive.
// If either are nil, then it is open on that side (nil, nil is the same as Iterate)
func (t *ImmutableTree) IterateRangeInclusive(start, end []byte, ascending bool, fn func(key, value []byte, version int64) bool) (stopped bool) {
	root := t.getRoot()
	if root == nil {
		return false
	}
	return root.traverseInRange(t, start, end, ascending, true, 0, func(node *Node, _ uint8) bool {
		if node.height == 0 {
			return fn(node.key, node.value, node.version)
		}
		return false
	})
}

// Clone creates a clone of the tree.
// Used internally by MutableTree.
func (t *ImmutableTree) clone() *ImmutableTree {
	return &ImmutableTree{
		root:          t.root,
		lastSavedRoot: t.lastSavedRoot,
		ndb:           t.ndb,
		version:       t.version,
		nodeVersions:  t.nodeVersions,
		isNotEmpty:    t.isNotEmpty,
	}
}

// nodeSize is like Size, but includes inner nodes too.
func (t *ImmutableTree) nodeSize() int {
	root := t.getRoot()
	if root == nil {
		return 0
	}
	size := 0
	root.traverse(t, true, func(n *Node) bool {
		size++
		return false
	})
	return size
}

func (t *ImmutableTree) memoryNodeSize() int {
	size := 0
	var iter func(*Node)
	iter = func(node *Node) {
		if node == nil {
			return
		}
		size++
		iter(node.leftNode)
		iter(node.rightNode)
	}
	iter(t.root)
	return size
}
