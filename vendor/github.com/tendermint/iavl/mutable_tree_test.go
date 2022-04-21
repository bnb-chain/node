package iavl

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/db"
)

func BenchmarkMutableTree_Set(b *testing.B) {
	db := db.NewDB("test", db.MemDBBackend, "")
	t := NewMutableTree(db, 100000)
	for i := 0; i < 1000000; i++ {
		t.Set(randBytes(10), []byte{})
	}
	b.ReportAllocs()
	runtime.GC()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		t.Set(randBytes(10), []byte{})
	}
}

func TestMutableTree_SetAndPrune(t *testing.T) {
	db := db.NewDB("test", db.MemDBBackend, "")
	tree := NewMutableTreeWithOpts(db, 0, 5, 5)
	tree.Set([]byte("k1"), []byte("v1"))
	PrintTreeByLevel(tree.ImmutableTree)
	_, version, err := tree.SaveVersion()
	require.NoError(t, err)
	require.Equal(t, int64(1), version)
	require.Equal(t, 1, tree.memoryNodeSize())

	tree.Set([]byte("k2"), []byte("v1"))
	PrintTreeByLevel(tree.ImmutableTree)
	_, version, err = tree.SaveVersion()
	require.NoError(t, err)
	require.Equal(t, int64(2), version)
	require.Equal(t, 3, tree.memoryNodeSize())

	tree.Set([]byte("k3"), []byte("v1"))
	PrintTreeByLevel(tree.ImmutableTree)
	_, version, err = tree.SaveVersion()
	require.NoError(t, err)
	require.Equal(t, int64(3), version)
	require.Equal(t, 5, tree.memoryNodeSize())

	tree.Set([]byte("k4"), []byte("v1"))
	PrintTreeByLevel(tree.ImmutableTree)
	_, version, err = tree.SaveVersion()
	require.NoError(t, err)
	require.Equal(t, int64(4), version)
	require.Equal(t, 7, tree.memoryNodeSize())

	tree.Set([]byte("k5"), []byte("v1"))
	PrintTreeByLevel(tree.ImmutableTree)
	_, version, err = tree.SaveVersion()
	require.NoError(t, err)
	require.Equal(t, int64(5), version)
	require.Equal(t, 9, tree.memoryNodeSize())

	tree.Set([]byte("k6"), []byte("v1"))
	tree.Set([]byte("k7"), []byte("v1"))
	tree.Set([]byte("k8"), []byte("v1"))
	tree.Set([]byte("k9"), []byte("v1"))
	tree.Set([]byte("k10"), []byte("v1"))
	PrintTreeByLevel(tree.ImmutableTree)
	_, version, err = tree.SaveVersion()
	require.NoError(t, err)
	require.Equal(t, int64(6), version)
	require.Equal(t, 19, tree.memoryNodeSize())

	tree.Set([]byte("k11"), []byte("v1"))
	PrintTreeByLevel(tree.ImmutableTree)
	_, version, err = tree.SaveVersion()
	require.NoError(t, err)
	require.Equal(t, int64(7), version)
	require.Equal(t, 11, tree.memoryNodeSize())
	PrintTreeByLevel(tree.ImmutableTree)
	fmt.Println()

	tree.Set([]byte("k12"), []byte("v1"))
	PrintTreeByLevel(tree.ImmutableTree)
	_, version, err = tree.SaveVersion()
	require.NoError(t, err)
	require.Equal(t, int64(8), version)
	fmt.Println()
	PrintTreeByLevel(tree.ImmutableTree)
}

func TestMutableTree_RemoveAndPrune(t *testing.T) {
	db := db.NewDB("test", db.MemDBBackend, "")
	tree := NewMutableTreeWithOpts(db, 0, 5, 5)
	tree.Set([]byte("k1"), []byte("v1"))
	tree.Set([]byte("k2"), []byte("v1"))
	tree.Set([]byte("k3"), []byte("v1"))
	tree.Set([]byte("k4"), []byte("v1"))
	tree.Set([]byte("k5"), []byte("v1"))
	//tree.Set([]byte("k6"), []byte("v1"))
	//tree.Set([]byte("k7"), []byte("v1"))
	//tree.Set([]byte("k8"), []byte("v1"))
	//tree.Set([]byte("k9"), []byte("v1"))
	PrintTreeByLevel(tree.ImmutableTree)
	tree.SaveVersion()
	PrintTreeByLevel(tree.ImmutableTree)
	tree.SaveVersion()
	PrintTreeByLevel(tree.ImmutableTree)
	tree.Remove([]byte("k5"))
	tree.SaveVersion()
	PrintTreeByLevel(tree.ImmutableTree)
	tree.Remove([]byte("k2"))
	PrintTreeByLevel(tree.ImmutableTree)
	tree.SaveVersion()
}
