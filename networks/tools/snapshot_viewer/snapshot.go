package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/go-amino"
	cryptoAmino "github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/libs/db"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/plugins/dex"
	"github.com/binance-chain/node/plugins/dex/order"
)

var codec = amino.NewCodec()

func init() {
	cryptoAmino.RegisterAmino(codec)
	dex.RegisterWire(codec)

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

func prepareCms(root string, appDB *db.GoLevelDB) sdk.CommitMultiStore {
	keys := []store.StoreKey{
		common.MainStoreKey, common.TokenStoreKey, common.DexStoreKey,
		common.PairStoreKey, common.GovStoreKey, common.StakeStoreKey,
		common.ParamsStoreKey, common.ValAddrStoreKey, common.AccountStoreKey,}

	cms := store.NewCommitMultiStore(appDB)
	for _, key := range keys {
		cms.MountStoreWithDB(key, sdk.StoreTypeIAVL, nil)
	}
	err := cms.LoadLatestVersion()
	if err != nil {
		fmt.Printf(err.Error())
		panic(err)
	}
	return cms
}

func genOrderBookSnapshotKeyPrefix(height int64) []byte {
	return []byte(fmt.Sprintf("orderbook_%v", height))
}

func genActiveOrdersSnapshotKey(height int64) []byte {
	return []byte(fmt.Sprintf("activeorders_%v", height))
}

func getSnapshot(height int64, root string) (obs map[string]order.OrderBookSnapshot, ao order.ActiveOrders) {
	db := openAppDB(root)
	defer db.Close()

	obs = make(map[string]order.OrderBookSnapshot)
	cms := prepareCms(root, db)
	orderbookKeyPrefix := genOrderBookSnapshotKeyPrefix(height)
	iter := sdk.KVStorePrefixIterator(cms.GetKVStore(common.DexStoreKey), orderbookKeyPrefix)
	defer iter.Close()
	var obSize int64 = 0
	for ; iter.Valid(); iter.Next() {
		fmt.Println(string(iter.Key()))
		obSize += int64(len(iter.Value()))
		bz := uncompress(iter.Value())
		if bz == nil {
			continue
		}
		var ob order.OrderBookSnapshot
		err := codec.UnmarshalBinaryLengthPrefixed(bz, &ob)
		if err != nil {
			panic(fmt.Sprintf("failed to unmarshal snapshort for orderbook [%s]", string(iter.Key())))
		}
		obs[string(iter.Key())] = ob
		fmt.Println(fmt.Sprintf("%#v", ob))
	}

	activeOrderKeyPrefix := genActiveOrdersSnapshotKey(height)
	bz := cms.GetKVStore(common.DexStoreKey).Get(activeOrderKeyPrefix)
	aoSize := len(bz)
	if bz == nil {
		return
	}
	bz = uncompress(bz)
	err := codec.UnmarshalBinaryLengthPrefixed(bz, &ao)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal snapshort for active orders [%s]", string(activeOrderKeyPrefix)))
	}
	//fmt.Println(fmt.Sprintf("%#v", ao))
	fmt.Println("active orders")
	for _, oi := range ao.Orders {
		fmt.Println(fmt.Sprintf("%#v", oi))
	}
	fmt.Println("order book size", obSize)
	fmt.Println("active order size", aoSize)
	return
}

func analyseSnapshot(height int64, home string) {
	fmt.Println("analysing...")
	obs, _ := getSnapshot(height, home)
	for symbol, ob := range obs {
		for _, p := range append(ob.Buys, ob.Sells...) {
			bug := false
			for i := 1; i < len(p.Orders); i++ {
				if p.Orders[i].Time < p.Orders[i-1].Time {
					bug = true
					break
				}
			}
			if bug {
				fmt.Println("!!!ob", symbol, p.Price, p.Orders)
			}
		}
	}

	//for _, order := range ao.Orders {
	//
	//}

}

func uncompress(bz []byte) []byte {
	b := bytes.NewReader(bz)
	var out bytes.Buffer
	r, _ := zlib.NewReader(b)
	defer r.Close()
	io.Copy(&out, r)
	return out.Bytes()
}

func printUsage() {
	fmt.Printf("usage: ./snapshot_viewer height home_path [--analysis]")
}

func main() {
	args := os.Args
	if len(args) != 3 && len(args) != 4 {
		printUsage()
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
		getSnapshot(height, home)
		return
	}

	arg3 := os.Args[3]
	if arg3 == "--analysis" {
		analyseSnapshot(height, home)
	} else {
		printUsage()
	}

}
