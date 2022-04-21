package iavl

// NOTE: This file favors int64 as opposed to int for size/counts.
// The Tree on the other hand favors int.  This is intentional.

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
)

// Node represents a node in a Tree.
type Node struct {
	key       []byte
	value     []byte
	version   int64
	height    int8
	size      int64
	hash      []byte
	leftHash  []byte
	leftNode  *Node
	rightHash []byte
	rightNode *Node
	persisted bool

	// version when the node is loaded into memory. we use the tree's current version.
	// as node.version always use tree.version+1 when it's created and persisted, the minimal loadVersion is node.version-1
	// we need to make sure node.loadVersion is always the largest version among all its children nodes
	loadVersion int64
	mtx sync.Mutex
}

// NewNode returns a new node from a key, value and version.
func NewNode(key []byte, value []byte, version int64) *Node {
	// the load version is tree's version, the node version is tree.version + 1
	return NewNodeWithLoadVersion(key, value, version, version-1)
}

func NewNodeWithLoadVersion(key []byte, value []byte, version int64, loadVersion int64) *Node {
	return &Node{
		key:         key,
		value:       value,
		height:      0,
		size:        1,
		version:     version,
		loadVersion: loadVersion,
	}
}

// MakeNode constructs an *Node from an encoded byte slice.
//
// The new node doesn't have its hash saved or set. The caller must set it
// afterwards.
func MakeNode(buf []byte) (*Node, cmn.Error) {

	// Read node header (height, size, version, key).
	height, n, cause := amino.DecodeInt8(buf)
	if cause != nil {
		return nil, cmn.ErrorWrap(cause, "decoding node.height")
	}
	buf = buf[n:]

	size, n, cause := amino.DecodeVarint(buf)
	if cause != nil {
		return nil, cmn.ErrorWrap(cause, "decoding node.size")
	}
	buf = buf[n:]

	ver, n, cause := amino.DecodeVarint(buf)
	if cause != nil {
		return nil, cmn.ErrorWrap(cause, "decoding node.version")
	}
	buf = buf[n:]

	key, n, cause := amino.DecodeByteSlice(buf)
	if cause != nil {
		return nil, cmn.ErrorWrap(cause, "decoding node.key")
	}
	buf = buf[n:]

	node := &Node{
		height:      height,
		size:        size,
		version:     ver,
		key:         key,
	}

	// Read node body.

	if node.isLeaf() {
		val, _, cause := amino.DecodeByteSlice(buf)
		if cause != nil {
			return nil, cmn.ErrorWrap(cause, "decoding node.value")
		}
		node.value = val
	} else { // Read children.
		leftHash, n, cause := amino.DecodeByteSlice(buf)
		if cause != nil {
			return nil, cmn.ErrorWrap(cause, "deocding node.leftHash")
		}
		buf = buf[n:]

		rightHash, _, cause := amino.DecodeByteSlice(buf)
		if cause != nil {
			return nil, cmn.ErrorWrap(cause, "decoding node.rightHash")
		}
		node.leftHash = leftHash
		node.rightHash = rightHash
	}
	return node, nil
}

// String returns a string representation of the node.
func (node *Node) String() string {
	hashstr := "<no hash>"
	if len(node.hash) > 0 {
		hashstr = fmt.Sprintf("%X", node.hash)
	}
	return fmt.Sprintf("Node{%s:%s@%d %X;%X}#%s",
		cmn.ColoredBytes(node.key, cmn.Green, cmn.Blue),
		cmn.ColoredBytes(node.value, cmn.Cyan, cmn.Blue),
		node.version,
		node.leftHash, node.rightHash,
		hashstr)
}

// clone creates a shallow copy of a node with its hash set to nil.
func (node *Node) clone(version int64) *Node {
	if node.isLeaf() {
		panic("Attempt to copy a leaf node")
	}
	return &Node{
		key:         node.key,
		height:      node.height,
		version:     version,
		size:        node.size,
		hash:        nil,
		leftHash:    node.leftHash,
		leftNode:    node.leftNode,
		rightHash:   node.rightHash,
		rightNode:   node.rightNode,
		persisted:   false,
		loadVersion: version - 1,
	}
}

func Key(node *Node) []byte   { return node.key }
func Value(node *Node) []byte { return node.value }

func IsLeaf(node *Node) bool { return node.isLeaf() }
func (node *Node) isLeaf() bool {
	return node.height == 0
}

// Check if the node has a descendant with the given key.
func (node *Node) has(t *ImmutableTree, key []byte) (has bool) {
	if bytes.Equal(node.key, key) {
		return true
	}
	if node.isLeaf() {
		return false
	}
	if bytes.Compare(key, node.key) < 0 {
		return node.getLeftNode(t, false).has(t, key)
	}
	return node.getRightNode(t, false).has(t, key)
}

// Get a key under the node.
func (node *Node) get(t *ImmutableTree, key []byte) (index int64, value []byte) {
	if node.isLeaf() {
		switch bytes.Compare(node.key, key) {
		case -1:
			return 1, nil
		case 1:
			return 0, nil
		default:
			return 0, node.value
		}
	}

	if bytes.Compare(key, node.key) < 0 {
		return node.getLeftNode(t, false).get(t, key)
	}
	rightNode := node.getRightNode(t, false)
	index, value = rightNode.get(t, key)
	index += node.size - rightNode.size
	return index, value
}

func (node *Node) getByIndex(t *ImmutableTree, index int64) (key []byte, value []byte) {
	if node.isLeaf() {
		if index == 0 {
			return node.key, node.value
		}
		return nil, nil
	}
	// TODO: could improve this by storing the
	// sizes as well as left/right hash.
	leftNode := node.getLeftNode(t, false)

	if index < leftNode.size {
		return leftNode.getByIndex(t, index)
	}
	return node.getRightNode(t, false).getByIndex(t, index-leftNode.size)
}

// Computes the hash of the node without computing its descendants. Must be
// called on nodes which have descendant node hashes already computed.
func Hash(node *Node) []byte { return node._hash() }
func (node *Node) _hash() []byte {
	if node.hash != nil {
		return node.hash
	}

	h := tmhash.New()
	buf := new(bytes.Buffer)
	if err := node.writeHashBytes(buf); err != nil {
		panic(err)
	}
	h.Write(buf.Bytes())
	node.hash = h.Sum(nil)

	return node.hash
}

// Hash the node and its descendants recursively. This usually mutates all
// descendant nodes. Returns the node hash and number of nodes hashed.
func (node *Node) hashWithCount() ([]byte, int64) {
	if node.hash != nil {
		return node.hash, 0
	}

	h := tmhash.New()
	buf := new(bytes.Buffer)
	hashCount, err := node.writeHashBytesRecursively(buf)
	if err != nil {
		panic(err)
	}
	h.Write(buf.Bytes())
	node.hash = h.Sum(nil)

	return node.hash, hashCount + 1
}

// Writes the node's hash to the given io.Writer. This function expects
// child hashes to be already set.
func (node *Node) writeHashBytes(w io.Writer) cmn.Error {
	err := amino.EncodeInt8(w, node.height)
	if err != nil {
		return cmn.ErrorWrap(err, "writing height")
	}
	err = amino.EncodeVarint(w, node.size)
	if err != nil {
		return cmn.ErrorWrap(err, "writing size")
	}
	err = amino.EncodeVarint(w, node.version)
	if err != nil {
		return cmn.ErrorWrap(err, "writing version")
	}

	// Key is not written for inner nodes, unlike writeBytes.

	if node.isLeaf() {
		err = amino.EncodeByteSlice(w, node.key)
		if err != nil {
			return cmn.ErrorWrap(err, "writing key")
		}
		// Indirection needed to provide proofs without values.
		// (e.g. proofLeafNode.ValueHash)
		valueHash := tmhash.Sum(node.value)
		err = amino.EncodeByteSlice(w, valueHash)
		if err != nil {
			return cmn.ErrorWrap(err, "writing value")
		}
	} else {
		if node.leftHash == nil || node.rightHash == nil {
			panic("Found an empty child hash")
		}
		err = amino.EncodeByteSlice(w, node.leftHash)
		if err != nil {
			return cmn.ErrorWrap(err, "writing left hash")
		}
		err = amino.EncodeByteSlice(w, node.rightHash)
		if err != nil {
			return cmn.ErrorWrap(err, "writing right hash")
		}
	}

	return nil
}

// Writes the node's hash to the given io.Writer.
// This function has the side-effect of calling hashWithCount.
func (node *Node) writeHashBytesRecursively(w io.Writer) (hashCount int64, err cmn.Error) {
	if node.leftNode != nil {
		leftHash, leftCount := node.leftNode.hashWithCount()
		node.leftHash = leftHash
		hashCount += leftCount
	}
	if node.rightNode != nil {
		rightHash, rightCount := node.rightNode.hashWithCount()
		node.rightHash = rightHash
		hashCount += rightCount
	}
	err = node.writeHashBytes(w)

	return
}

// the method is used to calculate the size needed by `writeBytes`
func (node *Node) aminoSize() int {
	// 1 is for the node.height
	n := 1 +
		amino.VarintSize(node.size) +
		amino.VarintSize(node.version) +
		amino.ByteSliceSize(node.key)
	if node.isLeaf() {
		n += amino.ByteSliceSize(node.value)
	} else {
		n += amino.ByteSliceSize(node.leftHash) +
			amino.ByteSliceSize(node.rightHash)
	}
	return n
}

// Writes the node as a serialized byte slice to the supplied io.Writer.
func (node *Node) writeBytes(w io.Writer) cmn.Error {
	var cause error
	cause = amino.EncodeInt8(w, node.height)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing height")
	}
	cause = amino.EncodeVarint(w, node.size)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing size")
	}
	cause = amino.EncodeVarint(w, node.version)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing version")
	}

	// Unlike writeHashBytes, key is written for inner nodes.
	cause = amino.EncodeByteSlice(w, node.key)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing key")
	}

	if node.isLeaf() {
		cause = amino.EncodeByteSlice(w, node.value)
		if cause != nil {
			return cmn.ErrorWrap(cause, "writing value")
		}
	} else {
		if node.leftHash == nil {
			panic("node.leftHash was nil in writeBytes")
		}
		cause = amino.EncodeByteSlice(w, node.leftHash)
		if cause != nil {
			return cmn.ErrorWrap(cause, "writing left hash")
		}

		if node.rightHash == nil {
			panic("node.rightHash was nil in writeBytes")
		}
		cause = amino.EncodeByteSlice(w, node.rightHash)
		if cause != nil {
			return cmn.ErrorWrap(cause, "writing right hash")
		}
	}
	return nil
}

func GetLeftNode(node *Node, t *ImmutableTree) *Node { return node.getLeftNode(t, false) }
// pass true to updateVersion only when you want to modify the tree.
// for read-only operations, we do not updateVersion.
func (node *Node) getLeftNode(t *ImmutableTree, updateVersion bool) *Node {
	if node.leftNode == nil {
		node.mtx.Lock()
		defer node.mtx.Unlock()
		if node.leftNode == nil {
			node.leftNode = t.ndb.GetNode(node.leftHash)
			if updateVersion {
				node.leftNode.loadVersion = t.version
			} else {
				// we need to make sure the loadVersion is always smaller than it's parent nodes.
				// so just use the minimal loadVersion(i.e. node.version-1)
				node.leftNode.loadVersion = node.leftNode.version - 1
			}
			t.nodeVersions.Inc1WithLock(node.leftNode.loadVersion)
			return node.leftNode
		}
	}
	if updateVersion {
		node.leftNode.updateLoadVersion(t)
	}
	return node.leftNode
}

func GetRightNode(node *Node, t *ImmutableTree) *Node { return node.getRightNode(t, false) }
// pass true to updateVersion only when you want to modify the tree.
// for read-only operations, we do not updateVersion.
func (node *Node) getRightNode(t *ImmutableTree, updateVersion bool) *Node {
	if node.rightNode == nil {
		node.mtx.Lock()
		defer node.mtx.Unlock()
		if node.rightNode == nil {
			node.rightNode = t.ndb.GetNode(node.rightHash)
			if updateVersion {
				node.rightNode.loadVersion = t.version
			} else {
				node.rightNode.loadVersion = node.rightNode.version - 1
			}
			t.nodeVersions.Inc1WithLock(node.rightNode.loadVersion)
			return node.rightNode
		}
	}
	if updateVersion {
		node.rightNode.updateLoadVersion(t)
	}
	return node.rightNode
}

// NOTE: mutates height and size
func (node *Node) calcHeightAndSize(t *ImmutableTree) {
	left, right := node.getLeftNode(t, true), node.getRightNode(t, true)
	node.height = maxInt8(left.height, right.height) + 1
	node.size = left.size + right.size
}

func (node *Node) calcBalance(t *ImmutableTree) int {
	return int(node.getLeftNode(t, true).height) - int(node.getRightNode(t, true).height)
}

func (node *Node) equals(other *Node) bool {
	if node == nil {
		return other == nil
	} else if other == nil {
		return false
	}
	// do not check loadVersion.
	return node.version == other.version &&
		node.size == other.size &&
		node.height == other.height &&
		bytes.Equal(node.hash, other.hash) &&
		bytes.Equal(node.key, other.key) &&
		bytes.Equal(node.value, other.value) &&
		bytes.Equal(node.leftHash, other.leftHash) &&
		bytes.Equal(node.rightHash, other.rightHash) &&
		node.persisted == other.persisted &&
		node.leftNode.equals(other.leftNode) &&
		node.rightNode.equals(other.rightNode)
}

// traverse is a wrapper over traverseInRange when we want the whole tree
func (node *Node) traverse(t *ImmutableTree, ascending bool, cb func(*Node) bool) bool {
	return node.traverseInRange(t, nil, nil, ascending, false, 0, func(node *Node, depth uint8) bool {
		return cb(node)
	})
}

// traverseFirst is a wrapper over traverseInRange when we want the whole tree and will traverse the leaf nodes
func (node *Node) traverseFirst(t *ImmutableTree, ascending bool, cb func(*Node) bool) bool {
	return node.traverseInRangeDiscardNode(t, nil, nil, ascending, false, 0, func(node *Node, depth uint8) bool {
		return cb(node)
	})
}

func (node *Node) traverseWithDepth(t *ImmutableTree, ascending bool, cb func(*Node, uint8) bool) bool {
	return node.traverseInRange(t, nil, nil, ascending, false, 0, cb)
}

func (node *Node) traverseInRange(t *ImmutableTree, start, end []byte, ascending bool, inclusive bool, depth uint8, cb func(*Node, uint8) bool) bool {
	afterStart := start == nil || bytes.Compare(start, node.key) < 0
	startOrAfter := start == nil || bytes.Compare(start, node.key) <= 0
	beforeEnd := end == nil || bytes.Compare(node.key, end) < 0
	if inclusive {
		beforeEnd = end == nil || bytes.Compare(node.key, end) <= 0
	}

	// Run callback per inner/leaf node.
	stop := false
	if !node.isLeaf() || (startOrAfter && beforeEnd) {
		stop = cb(node, depth)
		if stop {
			return stop
		}
	}
	if node.isLeaf() {
		return stop
	}

	if ascending {
		// check lower nodes, then higher
		if afterStart {
			stop = node.getLeftNode(t, false).traverseInRange(t, start, end, ascending, inclusive, depth+1, cb)
		}
		if stop {
			return stop
		}
		if beforeEnd {
			stop = node.getRightNode(t, false).traverseInRange(t, start, end, ascending, inclusive, depth+1, cb)
		}
	} else {
		// check the higher nodes first
		if beforeEnd {
			stop = node.getRightNode(t, false).traverseInRange(t, start, end, ascending, inclusive, depth+1, cb)
		}
		if stop {
			return stop
		}
		if afterStart {
			stop = node.getLeftNode(t, false).traverseInRange(t, start, end, ascending, inclusive, depth+1, cb)
		}
	}

	return stop
}

// This method doesn't hold any reference to loaded node, its cb's responsibility to decide whether hold Node reference
// If cb (like usage in state sync) don't need hold a reference to loaded Node, the memory footprint will be near to zero
func (node *Node) traverseInRangeDiscardNode(t *ImmutableTree, start, end []byte, ascending bool, inclusive bool, depth uint8, cb func(*Node, uint8) bool) bool {
	afterStart := start == nil || bytes.Compare(start, node.key) < 0
	startOrAfter := start == nil || bytes.Compare(start, node.key) <= 0
	beforeEnd := end == nil || bytes.Compare(node.key, end) < 0
	if inclusive {
		beforeEnd = end == nil || bytes.Compare(node.key, end) <= 0
	}

	// Run callback per inner/leaf node.
	stop := false
	if !node.isLeaf() || (startOrAfter && beforeEnd) {
		stop = cb(node, depth)
		if stop {
			node.leftNode = nil
			node.rightNode = nil
			return stop
		}
	}
	if node.isLeaf() {
		return stop
	}

	if ascending {
		// check lower nodes, then higher
		if afterStart {
			child := node.getLeftNode(t, false)
			node.leftNode = nil
			stop = child.traverseInRangeDiscardNode(t, start, end, ascending, inclusive, depth+1, cb)
		}
		if stop {
			return stop
		}
		if beforeEnd {
			child := node.getRightNode(t, false)
			node.rightNode = nil
			stop = child.traverseInRangeDiscardNode(t, start, end, ascending, inclusive, depth+1, cb)
		}
	} else {
		// check the higher nodes first
		if beforeEnd {
			child := node.getRightNode(t, false)
			node.rightNode = nil
			stop = child.traverseInRangeDiscardNode(t, start, end, ascending, inclusive, depth+1, cb)
		}
		if stop {
			return stop
		}
		if afterStart {
			child := node.getLeftNode(t, false)
			node.leftNode = nil
			stop = child.traverseInRangeDiscardNode(t, start, end, ascending, inclusive, depth+1, cb)
		}
	}

	return stop
}

// NOTE: not thread-safe
func (node *Node) updateLoadVersion(t *ImmutableTree) {
	if t != nil && node.loadVersion != t.version {
		t.nodeVersions.Update(node.loadVersion, t.version)
		node.loadVersion = t.version
	}
}

// Only used in testing...
func (node *Node) lmd(t *ImmutableTree) *Node {
	if node.isLeaf() {
		return node
	}
	return node.getLeftNode(t, false).lmd(t)
}
