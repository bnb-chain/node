package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/iavl"
	cryptoAmino "github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/state"
)

var codec = amino.NewCodec()

func init() {
	cryptoAmino.RegisterAmino(codec)
	types.RegisterWire(codec)

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("tbnb", "bnbp")
}

func openDB(root, dbName string) *db.GoLevelDB {
	db, err := db.NewGoLevelDB(dbName, path.Join(root, "data"))
	if err != nil {
		fmt.Printf("new levelDb err in path %s\n", path.Join(root, "data"))
		panic(err)
	}
	return db
}

func openAppDB(root string) *db.GoLevelDB {
	return openDB(root, "application")
}

func stateDiff(root1, root2 string) {
	s1 := getState(root1)
	s2 := getState(root2)
	fmt.Printf("State| Height:%d: AppHash:%X\n", s1.LastBlockHeight, s1.AppHash)
	fmt.Printf("State| Height:%d: AppHash:%X\n", s2.LastBlockHeight, s2.AppHash)
}

func getState(root string) state.State {
	stateDb := openDB(root, "state")
	defer stateDb.Close()
	return state.LoadState(stateDb)
}

func prepareCms(root string, appDB *db.GoLevelDB, storeKeys []store.StoreKey, height int64) sdk.CommitMultiStore {
	cms := store.NewCommitMultiStore(appDB)
	for _, key := range storeKeys {
		cms.MountStoreWithDB(key, sdk.StoreTypeIAVL, nil)
	}
	err := cms.LoadVersion(height)
	if err != nil {
		fmt.Printf("height %d does not exist in %s\n", height, root)
		panic(err)
	}
	return cms
}

func accountKeyDecoder(key []byte) string {
	prefix := []byte("account:")
	key = key[len(prefix):]
	return sdk.AccAddress(key).String()
}

func accountValueDecoder(value []byte) interface{} {
	acc := types.AppAccount{}
	codec.UnmarshalBinaryBare(value, &acc)
	return acc
}

func compare(height int64, root1, root2 string) {
	stateDiff(root1, root2)
	db1 := openAppDB(root1)
	defer db1.Close()
	db2 := openAppDB(root2)
	defer db2.Close()
	keys := []store.StoreKey{
		common.MainStoreKey, common.TokenStoreKey, common.DexStoreKey,
		common.PairStoreKey, common.GovStoreKey, common.StakeStoreKey,
		common.ParamsStoreKey, common.ValAddrStoreKey, common.AccountStoreKey}

	cms1 := prepareCms(root1, db1, keys, height)
	cms2 := prepareCms(root2, db2, keys, height)

	if bytes.Equal(cms1.LastCommitID().Hash, cms2.LastCommitID().Hash) {
		fmt.Printf("commitId is the same, %X", cms1.LastCommitID().Hash)
		return
	}

	fmt.Printf("ComId| Height:%d, AppHash:%X\n", cms1.LastCommitID().Version, cms1.LastCommitID().Hash)
	fmt.Printf("ComId| Height:%d, AppHash:%X\n", cms2.LastCommitID().Version, cms2.LastCommitID().Hash)
	for _, key := range keys {
		tree1 := cms1.GetCommitStore(key).(store.TreeStore).GetImmutableTree()
		tree2 := cms2.GetCommitStore(key).(store.TreeStore).GetImmutableTree()
		if bytes.Equal(tree1.Hash(), tree2.Hash()) {
			fmt.Printf("identical, %-6s, %X\n", key.Name(), tree1.Hash())
			continue
		}
		fmt.Printf("diff found in %s:\n", key.Name())
		// TODO: hardcode here, refactor later to support other keys
		if key == common.AccountStoreKey {
			diff(iavl.GetRoot(tree1), tree1, iavl.GetRoot(tree2), tree2, accountKeyDecoder, accountValueDecoder)
		} else {
			diff(iavl.GetRoot(tree1), tree1, iavl.GetRoot(tree2), tree2, nil, nil)
		}
	}
}

func diff(node1 *iavl.Node, tree1 *iavl.ImmutableTree,
	node2 *iavl.Node, tree2 *iavl.ImmutableTree,
	keyDecoder func([]byte) string, valueDecoder func([]byte) interface{}) {
	if node1 == nil && node2 == nil {
		return
	}

	if node1 == nil || node2 == nil {
		fmt.Printf("Diff found, node1: %v, node2: %v", node1, node2)
		return
	}

	if bytes.Equal(iavl.Hash(node1), iavl.Hash(node2)) {
		return
	}

	if iavl.IsLeaf(node1) && iavl.IsLeaf(node2) {
		if !bytes.Equal(iavl.Hash(node1), iavl.Hash(node2)) {
			fmt.Printf("\t%s\n<=> %s\n", node1, node2)
			fmt.Printf("\t%s\n<=> %s\n\n", nodeInfo(node1, keyDecoder, valueDecoder), nodeInfo(node2, keyDecoder, valueDecoder))
		}
		return
	} else if iavl.IsLeaf(node1) || iavl.IsLeaf(node2) {
		fmt.Println("node1 and node2 have different hierarchy")
		return
	} else {
		// ignore inner nodes
		// fmt.Printf("\t%s\n<=> %s\n", node1, node2)
	}
	diff(iavl.GetLeftNode(node1, tree1), tree1, iavl.GetLeftNode(node2, tree2), tree2, keyDecoder, valueDecoder)
	diff(iavl.GetRightNode(node1, tree1), tree1, iavl.GetRightNode(node2, tree2), tree2, keyDecoder, valueDecoder)
}

func nodeInfo(node *iavl.Node, keyDecoder func([]byte) string, valueDecoder func([]byte) interface{}) string {
	str := ""
	if keyDecoder != nil {
		key := keyDecoder(iavl.Key(node))
		str += fmt.Sprintf("key: %s", key)
	} else {
		str += fmt.Sprintf("key: %X", iavl.Key(node))
	}

	if valueDecoder != nil {
		value := valueDecoder(iavl.Value(node))
		str += fmt.Sprintf(", addr: %s", value.(types.AppAccount).Address.String())
		str += fmt.Sprintf(", value: %#v", value)
	} else {
		str += fmt.Sprintf(", value: %X", iavl.Value(node))
	}
	return str
}

func compareAccount(height int64, root1, root2, addr string) {
	db1 := openAppDB(root1)
	defer db1.Close()
	db2 := openAppDB(root2)
	defer db2.Close()
	keys := []store.StoreKey{
		common.MainStoreKey, common.TokenStoreKey, common.DexStoreKey,
		common.PairStoreKey, common.GovStoreKey, common.StakeStoreKey,
		common.ParamsStoreKey, common.ValAddrStoreKey, common.AccountStoreKey}

	cms1 := prepareCms(root1, db1, keys, height)
	cms2 := prepareCms(root2, db2, keys, height)

	a, _ := sdk.AccAddressFromBech32(addr)
	key := auth.AddressStoreKey(a)

	n1 := getNode(key, cms1)
	n2 := getNode(key, cms2)
	fmt.Println(n1)
	fmt.Println(n2)
	if n1 != nil {
		fmt.Println(accountKeyDecoder(iavl.Key(n1)))
		fmt.Printf("%#v\n", accountValueDecoder(iavl.Value(n1)))
	}
	if n2 != nil {
		fmt.Println(accountKeyDecoder(iavl.Key(n2)))
		fmt.Printf("%#v\n", accountValueDecoder(iavl.Value(n2)))
	}
}

func getNode(key []byte, cms sdk.CommitMultiStore) *iavl.Node {
	tree := cms.GetCommitStore(common.AccountStoreKey).(store.TreeStore).GetImmutableTree()
	rootNode := iavl.GetRoot(tree)

	var innerGetNode func(key []byte, node *iavl.Node, t *iavl.ImmutableTree) *iavl.Node
	innerGetNode = func(key []byte, node *iavl.Node, t *iavl.ImmutableTree) *iavl.Node {
		if iavl.IsLeaf(node) {
			if bytes.Compare(iavl.Key(node), key) != 0 {
				return nil
			} else {
				return node
			}
		}

		if bytes.Compare(key, iavl.Key(node)) < 0 {
			return innerGetNode(key, iavl.GetLeftNode(node, t), t)
		}
		return innerGetNode(key, iavl.GetRightNode(node, t), t)
	}
	return innerGetNode(key, rootNode, tree)
}

func main() {
	args := os.Args
	if len(args) != 4 && len(args) != 5 {
		fmt.Printf("usage: ./compare height home_path1 home_path2 [account_addr]")
		return
	}

	heightStr := os.Args[1]
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		fmt.Printf("parsing height[%s] error: %s", heightStr, err.Error())
		return
	}

	home1 := os.Args[2]
	home2 := os.Args[3]
	if len(os.Args) == 4 {
		compare(height, home1, home2)
		return
	}
	addr := os.Args[4]
	compareAccount(height, home1, home2, addr)
}
