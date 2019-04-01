package app

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	storePkg "github.com/cosmos/cosmos-sdk/store"

	"github.com/tendermint/iavl"
	abci "github.com/tendermint/tendermint/abci/types"
	bc "github.com/tendermint/tendermint/blockchain"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/utils"
)

type StateSyncManager struct {
	enabledStateSyncReactor bool // whether we enabledStateSyncReactor state sync reactor

	stateSyncHeight     int64
	stateSyncNumKeys    []int64
	stateSyncStoreInfos []storePkg.StoreInfo

	reloadingMtx sync.RWMutex // guard below fields to make sure no concurrent load snapshot and response snapshot, and they should be updated atomically

	stateCachedHeight int64
	numKeysCache      []int64
	totalKeyCache     int64 // cache sum of numKeysCache
	chunkCache        [][]byte
}

// Implement state sync related ABCI interfaces
func (app *BinanceChain) LatestSnapshot() (height int64, numKeys []int64, err error) {
	app.reloadingMtx.RLock()
	defer app.reloadingMtx.RUnlock()

	return app.stateCachedHeight, app.numKeysCache, nil
}

func (app *BinanceChain) ReadSnapshotChunk(height int64, startIndex, endIndex int64) (chunk [][]byte, err error) {
	app.reloadingMtx.RLock()
	defer app.reloadingMtx.RUnlock()

	app.Logger.Info("read snapshot chunk", "height", height, "startIndex", startIndex, "endIndex", endIndex)
	if height != app.stateCachedHeight {
		return nil, fmt.Errorf("peer requested a stale height we do not have, cacheHeight: %d", app.stateCachedHeight)
	}
	return app.chunkCache[startIndex:endIndex], nil
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
	app.Logger.Info("start write recovery chunk", "totalKeys", len(chunk))
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

		app.Logger.Debug("commit store", "store", storeName, "hash", nodeHash)
		nodeDB.Commit()
		iterated += storeKeys
	}

	// start serve other's state sync request
	app.reloadingMtx.Lock()
	defer app.reloadingMtx.Unlock()
	app.stateCachedHeight = app.stateSyncHeight
	app.numKeysCache = app.stateSyncNumKeys
	app.chunkCache = chunk

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
	app.resetDexKeeper(height)

	// init app cache
	accountStore := stores.GetKVStore(common.AccountStoreKey)
	app.SetAccountStoreCache(app.Codec, accountStore, app.baseConfig.AccountCacheSize)

	return nil
}

func (app BinanceChain) resetDexKeeper(height int64) {
	app.DexKeeper.ClearOrders()

	// TODO: figure out how to get block time here to get rid of time.Now() :(
	_, err := app.DexKeeper.LoadOrderBookSnapshot(app.CheckState.Ctx, height, time.Now(), app.baseConfig.BreatheBlockInterval, app.baseConfig.BreatheBlockDaysCountBack)
	if err != nil {
		panic(err)
	}
	app.DexKeeper.InitRecentPrices(app.CheckState.Ctx)

}

func (app *BinanceChain) initStateSyncManager(enabled bool) {
	app.enabledStateSyncReactor = enabled
	if enabled {
		height := app.getLastBreatheBlockHeight()
		go app.reloadSnapshot(height, false)
	}
}

// the method might take quite a while (> 5 seconds), BETTER to be called concurrently
// so we only do it once a day after breathe block
// we will refactor it into split chunks into snapshot file soon
func (app *BinanceChain) reloadSnapshot(height int64, retry bool) {
	if app.enabledStateSyncReactor {
		app.reloadingMtx.Lock()
		defer app.reloadingMtx.Unlock()

		app.latestSnapshotImpl(height, retry)
	}
}

func (app *BinanceChain) getLastBreatheBlockHeight() int64 {
	// we should only sync to breathe block height
	latestBlockHeight := app.LastBlockHeight()
	var timeOfLatestBlock time.Time
	if latestBlockHeight == 0 {
		timeOfLatestBlock = utils.Now()
	} else {
		blockDB := baseapp.LoadBlockDB()
		defer blockDB.Close()
		blockStore := bc.NewBlockStore(blockDB)
		block := blockStore.LoadBlock(latestBlockHeight)
		timeOfLatestBlock = block.Time
	}

	height := app.DexKeeper.GetLastBreatheBlockHeight(
		app.CheckState.Ctx,
		latestBlockHeight,
		timeOfLatestBlock,
		app.baseConfig.BreatheBlockInterval,
		app.baseConfig.BreatheBlockDaysCountBack)
	app.Logger.Info("get last breathe block height", "height", height)
	return height
}

func (app *BinanceChain) latestSnapshotImpl(height int64, retry bool) {
	defer func() {
		if r := recover(); r != nil {
			log := fmt.Sprintf("recovered: %v\nstack:\n%v", r, string(debug.Stack()))
			app.Logger.Error("failed loading latest snapshot", "err", log)
		}
	}()
	app.Logger.Info("reload latest snapshot", "height", height)

	failed := true
	for failed {
		failed = false
		totalKeys := int64(0)
		app.numKeysCache = make([]int64, 0, len(common.StoreKeyNames))
		app.chunkCache = make([][]byte, 0, app.totalKeyCache) // assuming we didn't increase too many account in a day

		for _, key := range common.StoreKeyNames {
			var storeKeys int64
			store := app.GetCommitMultiStore().GetKVStore(common.StoreKeyNameMap[key])
			mutableTree := store.(*storePkg.IavlStore).Tree
			if tree, err := mutableTree.GetImmutable(height); err == nil {
				tree.IterateFirst(func(nodeBytes []byte) {
					storeKeys++
					app.chunkCache = append(app.chunkCache, nodeBytes)
				})
			} else {
				app.Logger.Error("failed to load immutable tree", "err", err)
				failed = true
				time.Sleep(1 * time.Second) // Endblocker has notified this reload snapshot,
				// wait for 1 sec after commit finish
				if retry {
					break
				} else {
					return
				}
			}
			totalKeys += storeKeys
			app.numKeysCache = append(app.numKeysCache, storeKeys)
		}

		app.stateCachedHeight = height
		app.totalKeyCache = totalKeys
		app.Logger.Info("finish read snapshot chunk", "height", height, "keys", totalKeys)
	}
}
