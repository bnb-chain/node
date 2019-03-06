package app

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/server"
	storePkg "github.com/cosmos/cosmos-sdk/store"

	"github.com/tendermint/iavl"
	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/utils"
)

type StateSyncManager struct {
	stateSyncHeight     int64
	stateSyncNumKeys    []int64
	stateSyncStoreInfos []storePkg.StoreInfo
}

// Implement state sync related ABCI interfaces
func (app *BinanceChain) LatestSnapshot() (height int64, numKeys []int64, err error) {
	app.Logger.Info("query latest snapshot")
	numKeys = make([]int64, 0, len(common.StoreKeyNames))

	// we should only sync to breathe block height
	latestBlockHeight := app.LastBlockHeight()
	var timeOfLatestBlock time.Time
	if latestBlockHeight == 0 {
		timeOfLatestBlock = utils.Now()
	} else {
		block := server.BlockStore.LoadBlock(latestBlockHeight)
		timeOfLatestBlock = block.Time
	}

	height = app.DexKeeper.GetLastBreatheBlockHeight(
		app.CheckState.Ctx,
		latestBlockHeight,
		timeOfLatestBlock,
		app.baseConfig.BreatheBlockInterval,
		app.baseConfig.BreatheBlockDaysCountBack)

	for _, key := range common.StoreKeyNames {
		var storeKeys int64
		store := app.GetCommitMultiStore().GetKVStore(common.StoreKeyNameMap[key])
		mutableTree := store.(*storePkg.IavlStore).Tree
		if tree, err := mutableTree.GetImmutable(height); err == nil {
			tree.IterateFirst(func(nodeBytes []byte) {
				storeKeys++
			})
		} else {
			app.Logger.Error("failed to load immutable tree", "err", err)
		}
		numKeys = append(numKeys, storeKeys)
	}

	return
}

func (app *BinanceChain) ReadSnapshotChunk(height int64, startIndex, endIndex int64) (chunk [][]byte, err error) {
	app.Logger.Info("read snapshot chunk", "height", height, "startIndex", startIndex, "endIndex", endIndex)
	chunk = make([][]byte, 0, endIndex-startIndex)

	// TODO: can be optimized - direct jump to expected store
	iterated := int64(0)
	for _, key := range common.StoreKeyNames {
		store := app.GetCommitMultiStore().GetKVStore(common.StoreKeyNameMap[key])

		mutableTree := store.(*storePkg.IavlStore).Tree
		if tree, err := mutableTree.GetImmutable(height); err == nil {
			tree.IterateFirst(func(nodeBytes []byte) {
				if iterated >= startIndex && iterated < endIndex {
					chunk = append(chunk, nodeBytes)
				}
				iterated += 1
			})
		} else {
			app.Logger.Error("failed to load immutable tree", "err", err)
		}
	}

	if int64(len(chunk)) != (endIndex - startIndex) {
		app.Logger.Error("failed to load enough chunk", "expected", endIndex-startIndex, "got", len(chunk))
	}

	app.Logger.Info("finish read snapshot chunk", "height", height, "startIndex", startIndex, "endIndex", endIndex)
	return
}

func (app *BinanceChain) StartRecovery(height int64, numKeys []int64) error {
	app.Logger.Info("start recovery")
	app.stateSyncHeight = height
	app.stateSyncNumKeys = numKeys
	app.stateSyncStoreInfos = make([]storePkg.StoreInfo, 0)

	return nil
}

func (app *BinanceChain) WriteRecoveryChunk(chunk [][]byte) error {
	//store := app.GetCommitMultiStore().GetKVStore(common.StoreKeyNameMap[storeName])
	app.Logger.Info("start write recovery chunk")
	nodes := make([]*iavl.Node, 0)
	for idx := 0; idx < len(chunk); idx++ {
		node, _ := iavl.MakeNode(chunk[idx])
		iavl.Hash(node)
		nodes = append(nodes, node)
	}

	iterated := int64(0)
	for storeIdx, storeName := range common.StoreKeyNames {
		db := dbm.NewPrefixDB(app.GetDB(), []byte("s/k:"+storeName+"/"))
		nodeDB := iavl.NewNodeDB(db, 10000)

		var nodeHash []byte
		storeKeys := app.stateSyncNumKeys[storeIdx]
		if storeKeys > 0 {
			nodeDB.SaveRoot(nodes[iterated], app.stateSyncHeight, true)
			nodeHash = iavl.Hash(nodes[iterated])
			for i := int64(0); i < storeKeys; i++ {
				node := nodes[iterated+i]
				nodeDB.SaveNode(node)
			}
		} else {
			nodeDB.SaveEmptyRoot(app.stateSyncHeight, true)
			nodeHash = nil
		}

		app.stateSyncStoreInfos = append(app.stateSyncStoreInfos, storePkg.StoreInfo{
			Name: storeName,
			Core: storePkg.StoreCore{
				CommitID: storePkg.CommitID{
					Version: app.stateSyncHeight,
					Hash:    nodeHash,
				},
			},
		})

		app.Logger.Debug("commit store: %s, root hash: %X\n", storeName, nodeHash)
		nodeDB.Commit()
		iterated += storeKeys
	}

	app.Logger.Info("finished write recovery chunk")
	return nil
}

func (app *BinanceChain) EndRecovery(height int64) error {
	app.Logger.Info("finished recovery", "height", height)

	// simulate setLatestversion key
	batch := app.GetDB().NewBatch()
	latestBytes, _ := app.Codec.MarshalBinaryLengthPrefixed(height) // Does not error
	batch.Set([]byte("s/latest"), latestBytes)

	ci := storePkg.CommitInfo{
		Version:    height,
		StoreInfos: app.stateSyncStoreInfos,
	}
	cInfoBytes, err := app.Codec.MarshalBinaryLengthPrefixed(ci)
	if err != nil {
		panic(err)
	}
	cInfoKey := fmt.Sprintf("s/%d", height)
	batch.Set([]byte(cInfoKey), cInfoBytes)
	batch.WriteSync()

	// load into memory from db
	err = app.LoadCMSLatestVersion()
	if err != nil {
		cmn.Exit(err.Error())
	}
	stores := app.GetCommitMultiStore()
	commitId := stores.LastCommitID()
	hashHex := fmt.Sprintf("%X", commitId.Hash)
	app.Logger.Info("commit by state reactor", "version", commitId.Version, "hash", hashHex)

	// simulate we just "Commit()" :P
	app.SetCheckState(abci.Header{Height: height})
	app.DeliverState = nil

	// TODO: sync the breathe block on state sync and just call app.DexKeeper.Init() to recover order book and recentPrices to memory
	app.DexKeeper.InitRecentPrices(app.CheckState.Ctx)
	// TODO: figure out how to get block time here to get rid of time.Now() :(
	_, err = app.DexKeeper.LoadOrderBookSnapshot(app.CheckState.Ctx, height, time.Now(), app.baseConfig.BreatheBlockInterval, app.baseConfig.BreatheBlockDaysCountBack)
	if err != nil {
		panic(err)
	}

	// init app cache
	accountStore := stores.GetKVStore(common.AccountStoreKey)
	app.SetAccountStoreCache(app.Codec, accountStore, app.baseConfig.AccountCacheSize)

	return nil
}
