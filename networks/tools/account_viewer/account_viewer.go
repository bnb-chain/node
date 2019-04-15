package main

import (
	"bytes"
	"encoding/json"
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

func getState(root string) state.State {
	stateDb := openDB(root, "state")
	defer stateDb.Close()
	return state.LoadState(stateDb)
}

func prepareCms(root string, appDB *db.GoLevelDB, height int64) sdk.CommitMultiStore {
	keys := []store.StoreKey{
		common.MainStoreKey, common.TokenStoreKey, common.DexStoreKey,
		common.PairStoreKey, common.GovStoreKey, common.StakeStoreKey,
		common.ParamsStoreKey, common.ValAddrStoreKey, common.AccountStoreKey}

	cms := store.NewCommitMultiStore(appDB)
	for _, key := range keys {
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

func getAccountNumber(height int64, root string) {
	db := openAppDB(root)
	defer db.Close()

	cms := prepareCms(root, db, height)
	tree := cms.GetCommitStore(common.AccountStoreKey).(store.TreeStore).GetImmutableTree()
	var num int64 = 0
	var maxAccountNum int64 = 0
	var globalAccountNumber int64
	bz := cms.GetKVStore(common.AccountStoreKey).Get([]byte("globalAccountNumber"))
	if bz != nil {
		err := codec.UnmarshalBinaryLengthPrefixed(bz, &globalAccountNumber)
		if err != nil {
			panic(err)
		}
	}

	tree.Iterate(func(key []byte, value []byte) bool {
		if bytes.Compare([]byte("globalAccountNumber"), key) != 0 {
			num++
			accNum := accountValueDecoder(value).(types.AppAccount).AccountNumber
			if accNum > maxAccountNum {
				maxAccountNum = accNum
			}
		}
		return false
	})

	if num != maxAccountNum+1 || num != globalAccountNumber {
		fmt.Printf("total_account_number: %d, max_account_number: %d, global_account_number: %d\n", num, maxAccountNum, globalAccountNumber)
	}
	fmt.Printf("total account number: %d\n", num)
}

func getAccount(height int64, root, addr string) types.AppAccount {
	db := openAppDB(root)
	defer db.Close()

	cms := prepareCms(root, db, height)

	a, _ := sdk.AccAddressFromBech32(addr)
	key := auth.AddressStoreKey(a)

	n := getNode(key, cms)
	fmt.Println(n)
	if n != nil {
		return accountValueDecoder(iavl.Value(n)).(types.AppAccount)
	}
	return types.AppAccount{}
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

func getAccByNum(home string, height, targetAccNum int64) types.AppAccount {
	db := openAppDB(home)
	defer db.Close()

	cms := prepareCms(home, db, height)
	tree := cms.GetCommitStore(common.AccountStoreKey).(store.TreeStore).GetImmutableTree()
	var targetAcc types.AppAccount
	tree.Iterate(func(key []byte, value []byte) bool {
		acc := accountValueDecoder(value).(types.AppAccount)
		if acc.AccountNumber == targetAccNum {
			targetAcc = acc
			return true
		}
		return false
	})
	return targetAcc
}

func analysisAccByNum(height, accNum int64, home string) {
	var prevAccState types.AppAccount
	var currAccState types.AppAccount
	if height > 0 {
		prevAccState = getAccByNum(home, height-1, accNum)
		if prevAccState.Address == nil {
			fmt.Println(fmt.Sprintf("acc number %v does not exist", accNum))
			return
		}
	}

	currAccState = getAccount(height, home, prevAccState.Address.String())
	analysis(currAccState, prevAccState, height)
}

func analysisAcc(height int64, home, addr string) {
	var prevAccState types.AppAccount
	var currAccState types.AppAccount

	if height > 0 {
		prevAccState = getAccount(height-1, home, addr)
	}
	currAccState = getAccount(height, home, addr)
	analysis(currAccState, prevAccState, height)
}

func analysis(currAccState, prevAccState types.AppAccount, height int64) {
	printAccState(prevAccState, height-1)
	printAccState(currAccState, height)

	if prevAccState.Address == nil && currAccState.Address != nil {
		fmt.Printf("this account is newly created in height %d\n", height)
	} else if prevAccState.Address != nil && currAccState.Address == nil {
		fmt.Printf("WARNING!!! this account is lost in height %d\n", height)
	} else if prevAccState.Address == nil && currAccState.Address == nil {
		fmt.Printf("this account does not exist in height %d\n", height)
	} else {
		fmt.Println("=========diff=========")
		if prevAccState.Sequence != currAccState.Sequence {
			fmt.Printf("seq: %d => %d\n", prevAccState.Sequence, currAccState.Sequence)
		}
		prevAccState := normalizeAccCoins(prevAccState)
		currAccState := normalizeAccCoins(currAccState)
		if diff := currAccState.Coins.Minus(prevAccState.Coins); !diff.IsZero() {
			fmt.Printf("free balance: %#v\n", diff)
		}
		if diff := currAccState.FrozenCoins.Minus(prevAccState.FrozenCoins); !diff.IsZero() {
			fmt.Printf("frozen balance: %#v\n", diff)
		}
		if diff := currAccState.LockedCoins.Minus(prevAccState.LockedCoins); !diff.IsZero() {
			fmt.Printf("locked balance: %#v\n", diff)
		}

		// TODO: find all txs and trades that will influence the acc.
	}
}

func printAccState(accState types.AppAccount, height int64) {
	jsonValue, _ := json.Marshal(accState)
	fmt.Printf("%s@%d\n", accState.Address.String(), height)
	fmt.Printf("%s\n\n", string(jsonValue))
}

func normalizeAccCoins(acc types.AppAccount) types.AppAccount {
	if acc.Coins == nil {
		acc.Coins = sdk.Coins{}
	}

	if acc.FrozenCoins == nil {
		acc.FrozenCoins = sdk.Coins{}
	}

	if acc.LockedCoins == nil {
		acc.LockedCoins = sdk.Coins{}
	}

	return acc
}

func main() {
	args := os.Args
	if len(args) != 3 && len(args) != 4 {
		fmt.Printf("usage: ./account_viewer height home_path [account_addr|account_number]")
		return
	}

	heightStr := os.Args[1]
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		fmt.Printf("parsing height[%s] error: %s", heightStr, err.Error())
		return
	}

	home := os.Args[2]
	if len(os.Args) == 3 {
		getAccountNumber(height, home)
		return
	}
	arg3 := os.Args[3]
	if accNum, err := strconv.ParseInt(arg3, 10, 64); err == nil {
		analysisAccByNum(height, accNum, home)
	} else {
		analysisAcc(height, home, arg3)
	}

}
