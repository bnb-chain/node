package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/go/src/path"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/blockchain"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto/encoding/amino"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/state"

	"github.com/BiJie/BinanceChain/common"
)

var cdc = amino.NewCodec()

func init() {
	cryptoAmino.RegisterAmino(cdc)
}

func newLevelDb(id string, rootDir string) (db.DB, error) {
	config := cfg.DefaultConfig()
	config.DBBackend = "leveldb"
	config.DBPath = "data"
	config.RootDir = rootDir
	ctx := node.DBContext{
		ID:     id,
		Config: config,
	}
	dbIns, err := node.DefaultDBProvider(&ctx)
	return dbIns, err
}

func calcValidatorsKey(height int64) []byte {
	return []byte(cmn.Fmt("validatorsKey:%v", height))
}

func calcConsensusParamsKey(height int64) []byte {
	return []byte(cmn.Fmt("consensusParamsKey:%v", height))
}

func loadValidatorsInfo(db db.DB, height int64) *state.ValidatorsInfo {
	buf := db.Get(calcValidatorsKey(height))
	if len(buf) == 0 {
		return nil
	}

	v := new(state.ValidatorsInfo)
	err := cdc.UnmarshalBinaryBare(buf, v)
	if err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		cmn.Exit(cmn.Fmt(`LoadValidators: Data has been corrupted or its spec has changed:
                %v\n`, err))
	}
	// TODO: ensure that buf is completely read.

	return v
}

func loadConsensusParamsInfo(db db.DB, height int64) *state.ConsensusParamsInfo {
	buf := db.Get(calcConsensusParamsKey(height))
	if len(buf) == 0 {
		return nil
	}

	paramsInfo := new(state.ConsensusParamsInfo)
	err := cdc.UnmarshalBinaryBare(buf, paramsInfo)
	if err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		cmn.Exit(cmn.Fmt(`LoadConsensusParams: Data has been corrupted or its spec has changed:
                %v\n`, err))
	}
	// TODO: ensure that buf is completely read.

	return paramsInfo
}

func resetBlockChainState(height int64, rootDir string) {
	stateDb, err := newLevelDb("state", rootDir)
	if err != nil {
		fmt.Printf("new levelDb err in path %s\n", path.Join(rootDir, "data"))
		return
	}
	defer stateDb.Close()

	blockDb, err := newLevelDb("blockstore", rootDir)
	if err != nil {
		fmt.Printf("new levelDb err in path %s\n", path.Join(rootDir, "data"))
		return
	}
	defer blockDb.Close()

	bs := blockchain.NewBlockStore(blockDb)
	block := bs.LoadBlock(height + 1)

	lastValidators, _ := state.LoadValidators(stateDb, height)
	validators, _ := state.LoadValidators(stateDb, height+1)
	validatorInfo := loadValidatorsInfo(stateDb, height)

	lastConsensusParams, _ := state.LoadConsensusParams(stateDb, height+1)
	consensusInfo := loadConsensusParamsInfo(stateDb, height)

	blockState := state.State{
		ChainID:          block.ChainID,
		LastBlockHeight:  height,
		LastBlockTotalTx: block.TotalTxs - block.NumTxs,
		LastBlockID:      block.LastBlockID,
		LastBlockTime:    block.Time,

		Validators:                  validators,
		LastValidators:              lastValidators,
		LastHeightValidatorsChanged: validatorInfo.LastHeightChanged,

		ConsensusParams:                  lastConsensusParams,
		LastHeightConsensusParamsChanged: consensusInfo.LastHeightChanged,

		LastResultsHash: block.LastResultsHash,
		AppHash:         block.AppHash,
	}

	state.SaveState(stateDb, blockState)
}

func resetBlockStoreState(height int64, rootDir string) {
	blockDb, err := newLevelDb("blockstore", rootDir)
	if err != nil {
		fmt.Printf("new levelDb err in path %s\n", path.Join(rootDir, "data"))
		return
	}
	defer blockDb.Close()

	blockState := blockchain.LoadBlockStoreStateJSON(blockDb)
	blockState.Height = height
	blockState.Save(blockDb)
}

// Set the latest version.
func setLatestVersion(batch db.DB, version int64) {
	latestBytes, _ := cdc.MarshalBinary(version) // Does not error
	batch.Set([]byte("s/latest"), latestBytes)
}

func MountStoresIAVL(cms store.CommitMultiStore, keys ...*sdk.KVStoreKey) {
	for _, key := range keys {
		cms.MountStoreWithDB(key, sdk.StoreTypeIAVL, nil)
	}
}

func resetAppState(height int64, rootDir string) {
	dbIns, err := db.NewGoLevelDB("bnbchain", path.Join(rootDir, "data"))
	if err != nil {
		fmt.Printf("new levelDb err in path %s\n", path.Join(rootDir, "data"))
		return
	}
	defer dbIns.Close()

	cms := store.NewCommitMultiStore(dbIns)
	MountStoresIAVL(cms, common.MainStoreKey, common.AccountStoreKey, common.TokenStoreKey, common.DexStoreKey, common.PairStoreKey)
	cms.LoadLatestVersion()
	setLatestVersion(dbIns, height)
}

func resetAppVersionedTree(height int64, rootDir string) {
	dbIns, err := db.NewGoLevelDB("bnbchain", path.Join(rootDir, "data"))
	if err != nil {
		fmt.Printf("new levelDb err in path %s\n", path.Join(rootDir, "data"))
		return
	}
	defer dbIns.Close()

	keys := []store.StoreKey{common.MainStoreKey, common.AccountStoreKey, common.TokenStoreKey, common.DexStoreKey, common.PairStoreKey}

	for _, key := range keys {
		dbAccount := db.NewPrefixDB(dbIns, []byte("s/k:"+key.Name()+"/"))

		rootPrefixFmt := "r/%010d"
		for i := 1; i <= 100; i++ {
			rootKey := []byte(fmt.Sprintf(rootPrefixFmt, height+int64(i)))
			dbAccount.Delete(rootKey)
		}
	}
}

func resetPrivValidator(height int64, rootDir string) {
	privValidator := privval.LoadOrGenFilePV(path.Join(rootDir, "config/priv_validator.json"))
	privValidator.LastHeight = height
	privValidator.Save()
}

func getBlockChainHeight(rootDir string) int64 {
	blockDb, err := newLevelDb("blockstore", rootDir)
	if err != nil {
		fmt.Printf("new levelDb err in path %s\n", path.Join(rootDir, "data"))
		return -1
	}

	blockState := blockchain.LoadBlockStoreStateJSON(blockDb)
	return blockState.Height
}

func recoverBlockChain(height int64, rootDir string) {
	resetBlockChainState(height, rootDir)
	resetBlockStoreState(height, rootDir)
	resetAppState(height, rootDir)
	resetAppVersionedTree(height, rootDir)
	resetPrivValidator(height, rootDir)
}

// Purpose:
// 	Reset blockchain to a specific height and continue block from this height
//
// Usage:
// 	1. go build reset.go
// 	2. ./reset height_to_reset home_path1 home_path2 ...
func main() {
	args := os.Args
	if len(args) < 3 {
		fmt.Printf("usage: ./reset height home_path1 home_path2 ...")
		return
	}

	heightStr := os.Args[1]
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		fmt.Printf("parsing height[%s] error: %s", heightStr, err.Error())
		return
	}

	rootDirs := os.Args[2:]
	for _, dir := range rootDirs {
		fmt.Printf("rest home_path[%s] to height[%d]\n", dir, height)
		recoverBlockChain(height, dir)
	}
}
