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
	cfg "github.com/tendermint/tendermint/config"
	cryptoAmino "github.com/tendermint/tendermint/crypto/encoding/amino"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/state"
	tmstore "github.com/tendermint/tendermint/store"

	"github.com/bnb-chain/node/common"
)

var cdc = amino.NewCodec()

const latestStateToKeep int64 = 1 << 20

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

func calcStateKey(height int64) []byte {
	return []byte(fmt.Sprintf("stateKey:%v", height%latestStateToKeep))
}

func calcValidatorsKey(height int64) []byte {
	return []byte(fmt.Sprintf("validatorsKey:%v", height))
}

//nolint:deadcode,unused
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

func resetBlockChainState(height int64, rootDir string) {
	stateDb, err := newLevelDb("state", rootDir)
	if err != nil {
		cmn.Exit(fmt.Sprintf("new levelDb err in path %s\n", path.Join(rootDir, "data")))
		return
	}
	defer stateDb.Close()

	var blockState state.State
	latestState := state.LoadState(stateDb)
	latestHeight := latestState.LastBlockHeight
	if latestHeight-height > latestStateToKeep {
		blockDb, err := newLevelDb("blockstore", rootDir)
		if err != nil {
			cmn.Exit(fmt.Sprintf("new levelDb err in path %s\n", path.Join(rootDir, "data")))
			return
		}
		defer blockDb.Close()
		blockstore := tmstore.NewBlockStore(blockDb)
		block := blockstore.LoadBlock(height)
		nextBlock := blockstore.LoadBlock(height + 1)
		blockState = latestState.Copy()
		blockState.LastBlockHeight = height
		blockState.LastBlockTotalTx = block.TotalTxs
		blockState.LastBlockID = nextBlock.LastBlockID
		blockState.LastBlockTime = block.Time
		blockState.NextValidators, err = state.LoadValidators(stateDb, height+2)
		if err != nil {
			cmn.Exit("failed to load validator info")
			return
		}
		blockState.Validators, err = state.LoadValidators(stateDb, height+1)
		if err != nil {
			cmn.Exit("failed to load validator info")
			return
		}
		blockState.LastValidators, err = state.LoadValidators(stateDb, height)
		if err != nil {
			cmn.Exit("failed to load validator info")
			return
		}
		blockState.LastHeightConsensusParamsChanged = 1
		validatorInfo := loadValidatorsInfo(stateDb, height)
		if validatorInfo != nil {
			blockState.LastHeightValidatorsChanged = validatorInfo.LastHeightChanged
		}
		blockState.ConsensusParams, err = state.LoadConsensusParams(stateDb, height)
		if err != nil {
			cmn.Exit("failed to load consensusparam info")
			return
		}
		blockState.LastResultsHash = nextBlock.LastResultsHash
		blockState.AppHash = nextBlock.AppHash

	} else {
		buf := stateDb.Get(calcStateKey(height))
		err = cdc.UnmarshalBinaryBare(buf, &blockState)
		if err != nil {
			// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
			cmn.Exit(fmt.Sprintf(`LoadState: Data has been corrupted or its spec has changed:
                %v\n`, err))
		}
	}
	state.SaveState(stateDb, blockState)

	// reset index height in state db
	// NOTICE: Will not rollback index data in index db. If roll back all validators, the index db may contain dirty data.
	rawHeight := stateDb.Get(state.IndexHeightKey)
	if rawHeight != nil {
		var indexHeight int64
		err := cdc.UnmarshalBinaryBare(rawHeight, &indexHeight)
		if err != nil {
			// should not happen
			cmn.Exit(fmt.Sprintf(`Load IndexHeight: Data has been corrupted or its spec has changed:
                %v\n`, err))
		}
		if height < indexHeight {
			bz, err := cdc.MarshalBinaryBare(height)
			if err != nil {
				cmn.Exit(fmt.Sprintf(`Faile to marshal index height:
                %v\n`, err))
			} else {
				stateDb.Set(state.IndexHeightKey, bz)
			}
		}
	}
}

func resetBlockStoreState(height int64, rootDir string) {
	blockDb, err := newLevelDb("blockstore", rootDir)
	if err != nil {
		fmt.Printf("new levelDb err in path %s\n", path.Join(rootDir, "data"))
		return
	}
	defer blockDb.Close()

	blockState := tmstore.LoadBlockStoreStateJSON(blockDb)
	blockState.Height = height
	blockState.Save(blockDb)
}

// Set the latest version.
func setLatestVersion(batch db.DB, version int64) {
	latestBytes, _ := cdc.MarshalBinaryLengthPrefixed(version) // Does not error
	batch.Set([]byte("s/latest"), latestBytes)
}

//nolint:deadcode
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
	//cdc.UnmarshalBinaryLengthPrefixed(haha, &ha)
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

	keys := common.GetNonTransientStoreKeys()

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
				//fmt.Println("delete root version ", version, key.Name())
			}
		}
	}
}

func resetPrivValidator(height int64, rootDir string) {
	var privValidator *privval.FilePV
	keyPath := path.Join(rootDir, "config/priv_validator_key.json")
	statePath := path.Join(rootDir, "data/priv_validator_state.json")
	if cmn.FileExists(keyPath) && cmn.FileExists(statePath) {
		privValidator = privval.LoadFilePV(keyPath, statePath)
	} else {
		fmt.Printf("This is not a validator node, no need to reset priv_validator file")
		return
	}
	// TODO(#121): Should we also need reset LastRound, LastStep?
	privValidator.LastSignState.Height = height
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
		//fmt.Printf("delete orphan %v %x\n", height, itr.Value())
		dbAccount.Delete(itr.Key())
	}
}

// Purpose:
// 	Reset node to a specific height and continue block from this height
//
// Usage:
// 	1. go build state_recover.go
// 	2. ./state_recover height_to_reset home_path1 home_path2 ...

func printUsage() {
	fmt.Printf("usage: ./state_recover height home_path1 home_path2 ...\n")
}

func main() {
	args := os.Args
	if len(args) < 3 {
		printUsage()
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
		fmt.Printf("recover home_path[%s] to height[%d]\n", dir, height)
		restartNodeAtHeight(height, dir)
		fmt.Printf("recover success\n")
	}
}
