package iavl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNodeVersions_changes(t *testing.T) {
	nv := NewNodeVersions(10, 20, 0)
	nv.Inc(1, 10)
	require.Equal(t, 1, len(nv.changes))
	require.Equal(t, 10, nv.changes[1])
	nv.Inc1(1)
	require.Equal(t, 1, len(nv.changes))
	require.Equal(t, 11, nv.changes[1])
	nv.Dec(10, 5)
	require.Equal(t, 2, len(nv.changes))
	require.Equal(t, -5, nv.changes[10])
	require.Equal(t, 11, nv.changes[1])
	nv.Update(1, 20)
	require.Equal(t, 3, len(nv.changes))
	require.Equal(t, 10, nv.changes[1])
	require.Equal(t, -5, nv.changes[10])
	require.Equal(t, 1, nv.changes[20])
}

func TestNodeVersions_Commit(t *testing.T) {
	nv := NewNodeVersions(5, 5, 0)

	_, _, err := nv.Commit(20)
	require.Error(t, err)

	commitAndCheck(t, nv, 0, -5, 0)

	nv.Inc(1, 2)
	commitAndCheck(t, nv, 1, -4, 0)

	nv.Inc(2, 4)
	commitAndCheck(t, nv, 2, -3, 0)
	commitAndCheck(t, nv, 3, 1, 2)

	nv.Inc(3, 2)
	commitAndCheck(t, nv, 4, 2, 4)

	nv.Inc(4, 3)
	nv.Inc(5, 2)
	commitAndCheck(t, nv, 5, 0, 0)

	nv.Inc(1, 2)
	commitAndCheck(t, nv, 6, 3,4)
	// 4:3, 5:2

	nv.Inc(1, 3)
	nv.Inc(2, 5)
	commitAndCheck(t, nv, 7, 2, 8)

	nv.Inc(7, 0)
	commitAndCheck(t, nv, 8, 3, 0)
	commitAndCheck(t, nv, 9, 4, 3)
	commitAndCheck(t, nv, 10, 5, 2)
	commitAndCheck(t, nv, 11, 6, 0)
}

func commitAndCheck(t *testing.T, nv *NodeVersions, commitVersion int64, expectedMaxPruneVersion int64, expectedPruneNum int) {
	maxPruneVersion, pruneNum, err := nv.Commit(commitVersion)
	require.NoError(t, err)
	require.Equal(t, expectedMaxPruneVersion, maxPruneVersion)
	require.Equal(t, expectedPruneNum, pruneNum)
}