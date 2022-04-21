package iavl

import (
	"fmt"
	"sync"
)

// not Thread-Safe
type NodeVersions struct {
	nums    []int
	mtx     sync.Mutex
	changes map[int64]int // version -> num, changes will be merged to nums when commit

	maxVersions int
	maxNodes    int

	firstVersion   int64
	nextVersion    int64
	nextVersionIdx int

	totalNodes int
}

func NewNodeVersions(maxVersions int, maxNodes int, lastVersion int64) *NodeVersions {
	return &NodeVersions{
		nums:        make([]int, maxVersions, maxVersions),
		changes:     make(map[int64]int, 32),
		maxVersions: maxVersions,
		maxNodes:    maxNodes,

		firstVersion:   lastVersion - 1,
		nextVersion:    lastVersion,
		nextVersionIdx: 0,
		totalNodes:     0,
	}
}

func (nv *NodeVersions) Inc1(version int64) {
	nv.changes[version]++
}

func (nv *NodeVersions) Inc1WithLock(version int64) {
	nv.mtx.Lock()
	nv.changes[version]++
	nv.mtx.Unlock()
}

func (nv *NodeVersions) Inc(version int64, times int) {
	nv.changes[version] += times
}

func (nv *NodeVersions) Dec1(version int64) {
	nv.changes[version]--
}

func (nv *NodeVersions) Dec(version int64, times int) {
	nv.changes[version] -= times
}

func (nv *NodeVersions) Update(fromVersion, toVersion int64) {
	if fromVersion != toVersion {
		nv.changes[fromVersion]--
		nv.changes[toVersion]++
	}
}

func (nv *NodeVersions) Reset(tree *ImmutableTree) {
	nv.nums = make([]int, nv.maxVersions, nv.maxVersions)
	nv.changes = make(map[int64]int, 32)
	nv.nextVersionIdx = 0
	nv.totalNodes = 0

	if tree == nil {
		nv.firstVersion = 0
		nv.nextVersion = 1
		return
	}

	nv.firstVersion = tree.version - 1
	nv.nextVersion = tree.version

	var iter func(root *Node)
	iter = func(root *Node) {
		if root == nil {
			return
		}
		// root's version is the biggest in its branch.
		iter(root.leftNode)
		nv.Inc1(root.loadVersion)
		iter(root.rightNode)
	}
	iter(tree.root)
	for version, num := range nv.changes {
		idx := nv.getIndex(version)
		if idx < 0 {
			continue
		}
		nv.nums[idx] = num
		if version < nv.firstVersion {
			nv.firstVersion = version
		}
		nv.totalNodes += num
	}
	nv.changes = make(map[int64]int, len(nv.changes))
}

// we never prune nodes that have the same version with root node version.
func (nv *NodeVersions) Commit(newVersion int64) (maxPruneVersion int64, pruneNum int, err error) {
	if newVersion != nv.nextVersion {
		return 0, 0, fmt.Errorf("expect version %d, got %d", nv.nextVersion, newVersion)
	}

	olderVersionNums := 0
	for version, num := range nv.changes {
		if version > nv.nextVersion {
			// should not happen
			return 0, 0, fmt.Errorf("some changes happen on a future version %d, latest version is %d", version, newVersion)
		} else if version == nv.nextVersion {
			continue
			// skip it first, will handle this version later to avoid losing the original num of version (nv.nextVersion - nv.maxVersions)
		}
		if num == 0 {
			continue
		}

		versionIdx := nv.getIndex(version)
		if versionIdx < 0 {
			olderVersionNums += num
			continue
		}
		nv.nums[versionIdx] += num
		nv.totalNodes += num
		if version < nv.firstVersion {
			// some old version may be loaded in this round.
			nv.firstVersion = version
		}
	}
	maxPruneVersion, pruneNum = nv.prune()
	nv.nums[nv.nextVersionIdx] = nv.changes[nv.nextVersion]
	nv.totalNodes += nv.changes[nv.nextVersion] - pruneNum
	pruneNum += olderVersionNums

	nv.changes = make(map[int64]int, len(nv.changes))
	nv.nextVersion++
	nv.nextVersionIdx = nv.getNextIdx(nv.nextVersionIdx)
	nv.firstVersion = maxPruneVersion + 1
	return maxPruneVersion, pruneNum, nil
}

func (nv *NodeVersions) Rollback() {
	nv.changes = make(map[int64]int, len(nv.changes))
}

func (nv *NodeVersions) prune() (maxPruneVersion int64, prunedNum int) {
	if nv.totalNodes <= nv.maxNodes {
		return nv.nextVersion - int64(nv.maxVersions), nv.nums[nv.nextVersionIdx]
	}
	toPruneNum := nv.totalNodes - nv.maxNodes
	i := nv.getIndex(nv.firstVersion) // start from the idx of firstVersion to skip some zero nums.
	for {
		if nv.nums[i] > 0 {
			prunedNum += nv.nums[i]
			nv.nums[i] = 0
			if prunedNum >= toPruneNum { // the actual prune num would be equal or more than toPruneNum
				break
			}
		}
		i = nv.getNextIdx(i)
	}
	maxPruneVersion = nv.getVersion(i)
	return maxPruneVersion, prunedNum
}

func (nv *NodeVersions) getNextIdx(idx int) int {
	if idx < nv.maxVersions-1 {
		return idx + 1
	} else {
		return 0
	}
}

func (nv *NodeVersions) getIndex(version int64) int {
	if version < nv.nextVersion-int64(nv.maxVersions) {
		return -1
	}
	idx := nv.nextVersionIdx - int(nv.nextVersion-version)
	if idx < 0 {
		idx += nv.maxVersions
	}
	return idx
}

func (nv *NodeVersions) getVersion(idx int) int64 {
	if idx >= nv.nextVersionIdx {
		return nv.nextVersion - int64(nv.maxVersions) + int64(idx-nv.nextVersionIdx)
	}
	return nv.nextVersion - int64(nv.nextVersionIdx-idx)
}
