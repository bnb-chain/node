package iavl

import (
	"bytes"
	"fmt"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
)

const (
	defaultMaxVersions = 1000000
	defaultMaxNodes    = 750000
)

// ErrVersionDoesNotExist is returned if a requested version does not exist.
var ErrVersionDoesNotExist = fmt.Errorf("version does not exist")

// MutableTree is a persistent tree which keeps track of versions.
type MutableTree struct {
	*ImmutableTree // The current, working tree.

	orphans  map[string]int64 // Nodes removed by changes to working tree.
	versions map[int64]bool   // The previous, saved versions of the tree.
}

// NewMutableTree returns a new tree with the specified cache size and datastore.
func NewMutableTree(db dbm.DB, cacheSize int) *MutableTree {
	return NewMutableTreeWithOpts(db, cacheSize, defaultMaxVersions, defaultMaxNodes)
}

func NewMutableTreeWithOpts(db dbm.DB, cacheSize, maxVersions int, maxNodes int) *MutableTree {
	ndb := NewNodeDB(db, cacheSize)
	nodeVersions := NewNodeVersions(maxVersions, maxNodes, 0)
	head := &ImmutableTree{
		ndb:          ndb,
		nodeVersions: nodeVersions,
	}

	return &MutableTree{
		ImmutableTree: head,
		orphans:       map[string]int64{},
		versions:      map[int64]bool{},
	}
}

// IsEmpty returns whether or not the tree has any keys. Only trees that are
// not empty can be saved.
func (tree *MutableTree) IsEmpty() bool {
	return !tree.isNotEmpty
}

// VersionExists returns whether or not a version exists.
func (tree *MutableTree) VersionExists(version int64) bool {
	return tree.versions[version]
}

// Hash returns the hash of the latest saved version of the tree, as returned
// by SaveVersion. If no versions have been saved, Hash returns nil.
func (tree *MutableTree) Hash() []byte {
	if tree.version > 0 && tree.lastSavedRoot != nil {
		return tree.lastSavedRoot.hash
	}
	return nil
}

// WorkingHash returns the hash of the current working tree.
func (tree *MutableTree) WorkingHash() []byte {
	return tree.ImmutableTree.Hash()
}

// String returns a string representation of the tree.
func (tree *MutableTree) String() string {
	return tree.ndb.String()
}

// Set/Remove will orphan at most tree.Height nodes,
// balancing the tree after a Set/Remove will orphan at most 3 nodes.
func (tree *MutableTree) prepareOrphansSlice() []*Node {
	return make([]*Node, 0, tree.Height()+3)
}

// Set sets a key in the working tree. Nil values are not supported.
func (tree *MutableTree) Set(key, value []byte) bool {
	orphaned, updated := tree.set(key, value)
	tree.addOrphans(orphaned)
	return updated
}

func (tree *MutableTree) set(key []byte, value []byte) (orphans []*Node, updated bool) {
	if value == nil {
		panic(fmt.Sprintf("Attempt to store nil value at key '%s'", key))
	}
	if tree.ImmutableTree.getRoot() == nil {
		tree.ImmutableTree.root = NewNodeWithLoadVersion(key, value, tree.version+1, tree.version)
		tree.nodeVersions.Inc1(tree.root.loadVersion)
		tree.isNotEmpty = true
		return nil, false
	}
	orphans = tree.prepareOrphansSlice()
	tree.ImmutableTree.root, updated = tree.recursiveSet(tree.ImmutableTree.root, key, value, &orphans)
	return orphans, updated
}

func (tree *MutableTree) recursiveSet(node *Node, key []byte, value []byte, orphans *[]*Node) (
	newSelf *Node, updated bool,
) {
	version := tree.version + 1

	if node.isLeaf() {
		switch bytes.Compare(key, node.key) {
		case -1:
			tree.nodeVersions.Inc(tree.version, 2)
			return &Node{
				key:         node.key,
				height:      1,
				size:        2,
				leftNode:    NewNodeWithLoadVersion(key, value, version, tree.version),
				rightNode:   node,
				version:     version,
				loadVersion: tree.version,
			}, false
		case 1:
			tree.nodeVersions.Inc(tree.version, 2)
			return &Node{
				key:         key,
				height:      1,
				size:        2,
				leftNode:    node,
				rightNode:   NewNodeWithLoadVersion(key, value, version, tree.version),
				version:     version,
				loadVersion: tree.version,
			}, false
		default:
			*orphans = append(*orphans, node)
			tree.nodeVersions.Update(node.loadVersion, tree.version)
			return NewNodeWithLoadVersion(key, value, version, tree.version), true
		}
	} else {
		*orphans = append(*orphans, node)
		tree.nodeVersions.Update(node.loadVersion, tree.version)
		node = node.clone(version)

		if bytes.Compare(key, node.key) < 0 {
			node.leftNode, updated = tree.recursiveSet(node.getLeftNode(tree.ImmutableTree, true), key, value, orphans)
			node.leftHash = nil // leftHash is yet unknown
		} else {
			node.rightNode, updated = tree.recursiveSet(node.getRightNode(tree.ImmutableTree, true), key, value, orphans)
			node.rightHash = nil // rightHash is yet unknown
		}

		if updated {
			return node, updated
		}
		node.calcHeightAndSize(tree.ImmutableTree)
		newNode := tree.balance(node, orphans)
		return newNode, updated
	}
}

// Remove removes a key from the working tree.
func (tree *MutableTree) Remove(key []byte) ([]byte, bool) {
	val, orphaned, removed := tree.remove(key)
	tree.addOrphans(orphaned)
	return val, removed
}

// remove tries to remove a key from the tree and if removed, returns its
// value, nodes orphaned and 'true'.
func (tree *MutableTree) remove(key []byte) (value []byte, orphaned []*Node, removed bool) {
	if tree.getRoot() == nil {
		return nil, nil, false
	}
	orphaned = tree.prepareOrphansSlice()
	tree.root.updateLoadVersion(tree.ImmutableTree)
	newRootHash, newRoot, _, value := tree.recursiveRemove(tree.root, key, &orphaned)
	if len(orphaned) == 0 {
		return nil, nil, false
	}

	if newRoot == nil {
		if newRootHash != nil {
			tree.root = tree.ndb.GetNode(newRootHash)
			tree.root.loadVersion = tree.version
			tree.nodeVersions.Inc1(tree.root.loadVersion)
		} else {
			tree.root = nil
			tree.isNotEmpty = false
		}
	} else {
		tree.root = newRoot
	}
	return value, orphaned, true
}

// removes the node corresponding to the passed key and balances the tree.
// It returns:
// - the hash of the new node (or nil if the node is the one removed)
// - the node that replaces the orig. node after remove
// - new leftmost leaf key for tree after successfully removing 'key' if changed.
// - the removed value
// - the orphaned nodes.
func (tree *MutableTree) recursiveRemove(node *Node, key []byte, orphans *[]*Node) (newHash []byte, newSelf *Node, newKey []byte, newValue []byte) {
	version := tree.version + 1

	if node.isLeaf() {
		if bytes.Equal(key, node.key) {
			*orphans = append(*orphans, node)
			tree.nodeVersions.Dec1(node.loadVersion)
			return nil, nil, nil, node.value
		}
		return node.hash, node, nil, nil
	}

	// node.key < key; we go to the left to find the key:
	if bytes.Compare(key, node.key) < 0 {
		newLeftHash, newLeftNode, newKey, value := tree.recursiveRemove(node.getLeftNode(tree.ImmutableTree, true), key, orphans)
		if len(*orphans) == 0 {
			return node.hash, node, nil, value
		}
		*orphans = append(*orphans, node)
		if newLeftHash == nil && newLeftNode == nil { // left node held value, was removed
			tree.nodeVersions.Dec1(node.loadVersion)
			return node.rightHash, node.rightNode, node.key, value
		}
		tree.nodeVersions.Update(node.loadVersion, tree.version)
		newNode := node.clone(version)
		newNode.leftHash, newNode.leftNode = newLeftHash, newLeftNode
		newNode.calcHeightAndSize(tree.ImmutableTree)
		newNode = tree.balance(newNode, orphans)

		return newNode.hash, newNode, newKey, value
	}
	// node.key >= key; either found or look to the right:
	newRightHash, newRightNode, newKey, value := tree.recursiveRemove(node.getRightNode(tree.ImmutableTree, true), key, orphans)

	if len(*orphans) == 0 {
		return node.hash, node, nil, value
	}
	*orphans = append(*orphans, node)
	if newRightHash == nil && newRightNode == nil { // right node held value, was removed
		tree.nodeVersions.Dec1(node.loadVersion)
		return node.leftHash, node.leftNode, nil, value
	}
	tree.nodeVersions.Update(node.loadVersion, tree.version)
	newNode := node.clone(version)
	newNode.rightHash, newNode.rightNode = newRightHash, newRightNode
	if newKey != nil {
		newNode.key = newKey
	}
	newNode.calcHeightAndSize(tree.ImmutableTree)
	newNode = tree.balance(newNode, orphans)
	return newNode.hash, newNode, nil, value
}

// Load the latest versioned tree from disk.
func (tree *MutableTree) Load() (int64, error) {
	return tree.LoadVersion(int64(0))
}

// SetVersion set current version of the tree. Only used in upgrade
func (tree *MutableTree) SetVersion(version int64) {
	tree.version = version
	tree.ndb.latestVersion = version
	tree.nodeVersions.nextVersion = version
	tree.nodeVersions.firstVersion = version - 1
}

// Returns the version number of the latest version found
func (tree *MutableTree) LoadVersion(targetVersion int64) (int64, error) {
	roots, err := tree.ndb.getRoots()
	if err != nil {
		return 0, err
	}
	if len(roots) == 0 {
		return 0, nil
	}
	latestVersion := int64(0)
	var latestRoot []byte
	for version, r := range roots {
		tree.versions[version] = true
		if version > latestVersion &&
			(targetVersion == 0 || version <= targetVersion) {
			latestVersion = version
			latestRoot = r
		}
	}

	if !(targetVersion == 0 || latestVersion == targetVersion) {
		return latestVersion, fmt.Errorf("wanted to load target %v but only found up to %v",
			targetVersion, latestVersion)
	}

	nodeVersions := NewNodeVersions(tree.nodeVersions.maxVersions, tree.nodeVersions.maxNodes, latestVersion)
	t := &ImmutableTree{
		ndb:          tree.ndb,
		version:      latestVersion,
		nodeVersions: nodeVersions,
	}
	if len(latestRoot) != 0 {
		t.root = tree.ndb.GetNode(latestRoot)
		t.root.loadVersion = latestVersion
		t.lastSavedRoot = t.root
		t.nodeVersions.Inc1(t.root.loadVersion)
		t.isNotEmpty = true
	}

	tree.orphans = map[string]int64{}
	tree.ImmutableTree = t
	return latestVersion, nil
}

// LoadVersionOverwrite returns the version number of targetVersion.
// Higher versions' data will be deleted.
func (tree *MutableTree) LoadVersionForOverwriting(targetVersion int64) (int64, error) {
	latestVersion, err := tree.LoadVersion(targetVersion)
	if err != nil {
		return latestVersion, err
	}
	tree.deleteVersionsFrom(targetVersion + 1)
	return targetVersion, nil
}

// GetImmutable loads an ImmutableTree at a given version for querying
func (tree *MutableTree) GetImmutable(version int64) (*ImmutableTree, error) {
	rootHash := tree.ndb.getRoot(version)
	if rootHash == nil {
		return nil, ErrVersionDoesNotExist
	} else if len(rootHash) == 0 {
		return &ImmutableTree{
			ndb:          tree.ndb,
			version:      version,
			nodeVersions: NewNodeVersions(tree.nodeVersions.maxVersions, tree.nodeVersions.maxNodes, version),
		}, nil
	}
	nv := NewNodeVersions(tree.nodeVersions.maxVersions, tree.nodeVersions.maxNodes, version)
	root := tree.ndb.GetNode(rootHash)
	root.loadVersion = version
	nv.Inc1(root.loadVersion)
	return &ImmutableTree{
		root:         root,
		lastSavedRoot:root,
		ndb:          tree.ndb,
		version:      version,
		nodeVersions: nv,
		isNotEmpty:   true,
	}, nil
}

// Rollback resets the working tree to the latest saved version, discarding
// any unsaved modifications.
func (tree *MutableTree) Rollback() {
	if tree.version > 0 {
		tree.ImmutableTree.root = tree.lastSavedRoot
	} else {
		tree.ImmutableTree = &ImmutableTree{ndb: tree.ndb, version: 0, nodeVersions: tree.nodeVersions}
	}
	tree.nodeVersions.Rollback()
	tree.orphans = map[string]int64{}
}

// GetVersioned gets the value at the specified key and version.
func (tree *MutableTree) GetVersioned(key []byte, version int64) (
	index int64, value []byte,
) {
	if tree.versions[version] {
		t, err := tree.GetImmutable(version)
		if err != nil {
			return -1, nil
		}
		return t.Get(key)
	}
	return -1, nil
}

// SaveVersion saves a new tree version to disk, based on the current state of
// the tree. Returns the hash and new version number.
func (tree *MutableTree) SaveVersion() ([]byte, int64, error) {
	version := tree.version + 1

	if tree.versions[version] {
		//version already exists, throw an error if attempting to overwrite
		// Same hash means idempotent.  Return success.
		existingHash := tree.ndb.getRoot(version)
		var newHash = tree.WorkingHash()
		if bytes.Equal(existingHash, newHash) {
			tree.version = version
			tree.lastSavedRoot = tree.root
			tree.ImmutableTree = tree.ImmutableTree.clone()
			tree.orphans = map[string]int64{}
			tree.nodeVersions.Reset(tree.ImmutableTree)
			return existingHash, version, nil
		}
		return nil, version, fmt.Errorf("version %d was already saved to different hash %X (existing hash %X)",
			version, newHash, existingHash)
	}

	if tree.IsEmpty() {
		// There can still be orphans, for example if the root is the node being
		// removed.
		debug("SAVE EMPTY TREE %v\n", version)
		tree.ndb.SaveOrphans(version, tree.orphans)
		tree.ndb.SaveEmptyRoot(version, false)
	} else {
		debug("SAVE TREE %v\n", version)
		// Save the current tree.
		tree.ndb.SaveBranch(tree.getRoot())
		tree.ndb.SaveOrphans(version, tree.orphans)
		tree.ndb.SaveRoot(tree.getRoot(), version, false)
	}
	tree.ndb.Commit()
	// set lastSaveRoot before pruning, we ensure lastSavedRoot is not nil if the tree is not empty
	tree.updateLastSaveRoot()
	maxPruneVersion, pruneNum, err := tree.nodeVersions.Commit(tree.version)
	if err != nil {
		return nil, version, err
	}
	if pruneNum > 0 {
		tree.PruneInMemory(maxPruneVersion)
	}

	// Set new working tree.
	tree.version = version
	tree.versions[version] = true
	tree.ImmutableTree = tree.ImmutableTree.clone()
	tree.orphans = map[string]int64{}
	return tree.Hash(), version, nil
}

func (tree *MutableTree) PruneInMemory(maxPruneVersion int64) {
	if tree.root == nil {
		return
	}
	// root node can also be pruned from memory.
	if tree.root.loadVersion <= maxPruneVersion {
		tree.root = nil
	}

	var iter func(root *Node)
	iter = func(root *Node) {
		if root == nil {
			return
		}
		// root's version is the biggest in its branch.
		if left := root.leftNode; left != nil {
			if left.loadVersion <= maxPruneVersion {
				root.leftNode = nil
			} else {
				iter(left)
			}
		}
		if right := root.rightNode; right != nil {
			if right.loadVersion <= maxPruneVersion {
				root.rightNode = nil
			} else {
				iter(right)
			}
		}
	}
	iter(tree.ImmutableTree.root)
}

// DeleteVersion deletes a tree version from disk. The version can then no
// longer be accessed.
func (tree *MutableTree) DeleteVersion(version int64) error {
	if version == 0 {
		return cmn.NewError("version must be greater than 0")
	}
	if version == tree.version {
		return cmn.NewError("cannot delete latest saved version (%d)", version)
	}
	if _, ok := tree.versions[version]; !ok {
		return cmn.ErrorWrap(ErrVersionDoesNotExist, "")
	}

	tree.ndb.DeleteVersion(version, true)
	tree.ndb.Commit()

	delete(tree.versions, version)

	return nil
}

// deleteVersionsFrom deletes tree version from disk specified version to latest version. The version can then no
// longer be accessed.
func (tree *MutableTree) deleteVersionsFrom(version int64) error {
	if version <= 0 {
		return cmn.NewError("version must be greater than 0")
	}
	newLatestVersion := version - 1
	lastestVersion := tree.ndb.getLatestVersion()
	for ; version <= lastestVersion; version++ {
		if version == tree.version {
			return cmn.NewError("cannot delete latest saved version (%d)", version)
		}
		if _, ok := tree.versions[version]; !ok {
			return cmn.ErrorWrap(ErrVersionDoesNotExist, "")
		}
		tree.ndb.DeleteVersion(version, false)
		delete(tree.versions, version)
	}
	tree.ndb.Commit()
	tree.ndb.resetLatestVersion(newLatestVersion)
	return nil
}

// Rotate right and return the new node and orphan.
func (tree *MutableTree) rotateRight(node *Node, orphans *[]*Node) *Node {
	version := tree.version + 1

	// TODO: optimize balance & rotate.
	*orphans = append(*orphans, node)
	tree.nodeVersions.Update(node.loadVersion, tree.version)
	node = node.clone(version)

	orphaned := node.getLeftNode(tree.ImmutableTree, false)
	*orphans = append(*orphans, orphaned)
	tree.nodeVersions.Update(orphaned.loadVersion, tree.version)
	newNode := orphaned.clone(version)

	newNoderHash, newNoderCached := newNode.rightHash, newNode.rightNode
	newNode.rightHash, newNode.rightNode = node.hash, node
	node.leftHash, node.leftNode = newNoderHash, newNoderCached
	node.calcHeightAndSize(tree.ImmutableTree)
	newNode.calcHeightAndSize(tree.ImmutableTree)

	return newNode
}

// Rotate left and return the new node and orphan.
func (tree *MutableTree) rotateLeft(node *Node, orphans *[]*Node) *Node {
	version := tree.version + 1

	// TODO: optimize balance & rotate.
	*orphans = append(*orphans, node)
	tree.nodeVersions.Update(node.loadVersion, tree.version)
	node = node.clone(version)

	orphaned := node.getRightNode(tree.ImmutableTree, false)
	*orphans = append(*orphans, orphaned)
	tree.nodeVersions.Update(orphaned.loadVersion, tree.version)
	newNode := orphaned.clone(version)

	newNodelHash, newNodelCached := newNode.leftHash, newNode.leftNode
	newNode.leftHash, newNode.leftNode = node.hash, node
	node.rightHash, node.rightNode = newNodelHash, newNodelCached

	node.calcHeightAndSize(tree.ImmutableTree)
	newNode.calcHeightAndSize(tree.ImmutableTree)

	return newNode
}

// NOTE: assumes that node can be modified
// TODO: optimize balance & rotate
func (tree *MutableTree) balance(node *Node, orphans *[]*Node) (newSelf *Node) {
	if node.persisted {
		panic("Unexpected balance() call on persisted node")
	}
	balance := node.calcBalance(tree.ImmutableTree)

	if balance > 1 {
		left := node.getLeftNode(tree.ImmutableTree, true)
		if left.calcBalance(tree.ImmutableTree) >= 0 {
			// Left Left Case
			newNode := tree.rotateRight(node, orphans)
			return newNode
		}
		// Left Right Case
		node.leftHash = nil
		node.leftNode = tree.rotateLeft(left, orphans)
		newNode := tree.rotateRight(node, orphans)
		return newNode
	}
	if balance < -1 {
		right := node.getRightNode(tree.ImmutableTree, true)
		if right.calcBalance(tree.ImmutableTree) <= 0 {
			// Right Right Case
			newNode := tree.rotateLeft(node, orphans)
			return newNode
		}
		// Right Left Case
		node.rightHash = nil
		node.rightNode = tree.rotateRight(right, orphans)
		newNode := tree.rotateLeft(node, orphans)
		return newNode
	}
	// Nothing changed
	return node
}

func (tree *MutableTree) addOrphans(orphans []*Node) {
	for _, node := range orphans {
		if !node.persisted {
			// We don't need to orphan nodes that were never persisted.
			continue
		}
		if len(node.hash) == 0 {
			panic("Expected to find node hash, but was empty")
		}
		tree.orphans[string(node.hash)] = node.version
	}
}
