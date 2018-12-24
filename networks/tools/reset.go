package main

import (
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/iavl"
	"github.com/tendermint/tendermint/blockchain"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto/encoding/amino"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	dbm "github.com/tendermint/tendermint/libs/db"
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
	return []byte(fmt.Sprintf("validatorsKey:%v", height))
}

func calcConsensusParamsKey(height int64) []byte {
	return []byte(fmt.Sprintf("consensusParamsKey:%v", height))
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
		cmn.Exit(fmt.Sprintf(`LoadValidators: Data has been corrupted or its spec has changed:
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
		cmn.Exit(fmt.Sprintf(`LoadConsensusParams: Data has been corrupted or its spec has changed:
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
	previousBlock := bs.LoadBlock(height)

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
		LastBlockTime:    previousBlock.Time,

		NextValidators:              validators,
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
	dbIns, err := db.NewGoLevelDB("application", path.Join(rootDir, "data"))
	if err != nil {
		fmt.Printf("new levelDb err in path %s\n", path.Join(rootDir, "data"))
		return
	}
	defer dbIns.Close()

	//cms := store.NewCommitMultiStore(dbIns)
	//MountStoresIAVL(cms, common.MainStoreKey, common.AccountStoreKey, common.TokenStoreKey, common.DexStoreKey, common.PairStoreKey)
	//cms.LoadLatestVersion()
	//haha := dbIns.Get([]byte("s/latest"))
	//var ha int64
	//cdc.UnmarshalBinary(haha, &ha)
	//fmt.Printf("%x", haha)
	setLatestVersion(dbIns, height)
}

func resetAppVersionedTree(height int64, rootDir string) {
	dbIns, err := db.NewGoLevelDB("application", path.Join(rootDir, "data"))
	if err != nil {
		fmt.Printf("new levelDb err in path %s\n", path.Join(rootDir, "data"))
		return
	}
	defer dbIns.Close()

	keys := []store.StoreKey{common.MainStoreKey, common.AccountStoreKey, common.TokenStoreKey, common.DexStoreKey, common.PairStoreKey, common.GovStoreKey,
		common.StakeStoreKey, common.ParamsStoreKey, common.ValAddrStoreKey}

	for _, key := range keys {
		dbAccount := db.NewPrefixDB(dbIns, []byte("s/k:"+key.Name()+"/"))
		roots, err := getRoots(dbIns, key)
		if err != nil {
			panic("get roots error")
		}

		for version := range roots {
			if version >= height {
				deleteOrphans(dbIns, key, version)
			}

			if version > height {
				rootPrefixFmt := iavl.NewKeyFormat('r', 8)
				dbAccount.Delete(rootPrefixFmt.Key(version))
				fmt.Println("delete root version ", version, key.Name())
			}
		}
	}
}

func resetPrivValidator(height int64, rootDir string) {
	var privValidator *privval.FilePV
	filePath := path.Join(rootDir, "config/priv_validator.json")
	if cmn.FileExists(filePath) {
		privValidator = privval.LoadFilePV(filePath)
	} else {
		fmt.Printf("This is not a validator node, no need to reset priv_validator file")
	}
	// TODO(#121): Should we also need reset LastRound, LastStep?
	privValidator.LastHeight = height
	privValidator.Save()
}

func restartNodeAtHeight(height int64, rootDir string) {
	resetBlockChainState(height, rootDir)
	resetBlockStoreState(height, rootDir)
	resetAppState(height, rootDir)
	resetAppVersionedTree(height, rootDir)
	resetPrivValidator(height, rootDir)
}

func getRoots(dbIns *db.GoLevelDB, storeKey store.StoreKey) (map[int64][]byte, error) {
	roots := map[int64][]byte{}
	rootKeyFormat := iavl.NewKeyFormat('r', 8)

	prefixDB := db.NewPrefixDB(dbIns, []byte("s/k:"+storeKey.Name()+"/"))

	itr := dbm.IteratePrefix(prefixDB, rootKeyFormat.Key())
	defer itr.Close()

	for ; itr.Valid(); itr.Next() {
		var version int64
		rootKeyFormat.Scan(itr.Key(), &version)
		roots[version] = itr.Value()
	}

	return roots, nil
}

func deleteOrphans(dbIns *db.GoLevelDB, storeKey store.StoreKey, height int64) {
	nodeKeyFormat := iavl.NewKeyFormat('o', 8, 8, 20)
	dbAccount := db.NewPrefixDB(dbIns, []byte("s/k:"+storeKey.Name()+"/"))
	itr := dbm.IteratePrefix(dbAccount, nodeKeyFormat.Key(height))
	defer itr.Close()

	for ; itr.Valid(); itr.Next() {
		fmt.Printf("delete orphan %v %x\n", height, itr.Value())
		dbAccount.Delete(itr.Key())
	}
}

// Purpose:
// 	Reset node to a specific height and continue block from this height
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
		restartNodeAtHeight(height, dir)
	}
}
