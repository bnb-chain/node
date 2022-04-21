package types

import "fmt"

var UpgradeMgr = NewUpgradeManager(UpgradeConfig{})

const (
	FixSignBytesOverflow = "FixSignBytesOverflow" // fix json unmarshal overflow when build SignBytes
	BEP9                 = "BEP9"                 // https://github.com/binance-chain/BEPs/pull/9
	BEP12                = "BEP12"                // https://github.com/binance-chain/BEPs/pull/17
	BEP3                 = "BEP3"                 // https://github.com/binance-chain/BEPs/pull/30
	BEP8                 = "BEP8"                 // https://github.com/binance-chain/BEPs/pull/69
	LaunchBscUpgrade     = "LaunchBscUpgrade"
	BEP82                = "BEP82"                // https://github.com/binance-chain/BEPs/pull/82
	FixFailAckPackage    = "FixFailAckPackage"
	BEP128               = "BEP128" 			  //https://github.com/binance-chain/BEPs/pull/128
)

var MainNetConfig = UpgradeConfig{
	HeightMap: map[string]int64{},
}

type UpgradeConfig struct {
	HeightMap     map[string]int64
	StoreKeyMap   map[string]int64
	MsgTypeMap    map[string]int64
	BeginBlockers map[int64][]func(ctx Context)
}

type UpgradeManager struct {
	Config UpgradeConfig
	Height int64
}

func NewUpgradeManager(config UpgradeConfig) *UpgradeManager {
	return &UpgradeManager{
		Config: config,
	}
}

func (mgr *UpgradeManager) AddConfig(config UpgradeConfig) {
	for name, height := range config.HeightMap {
		mgr.AddUpgradeHeight(name, height)
	}
}

func (mgr *UpgradeManager) SetHeight(height int64) {
	mgr.Height = height
}

func (mgr *UpgradeManager) GetHeight() int64 {
	return mgr.Height
}

// run in every ABCI BeginBlock.
func (mgr *UpgradeManager) BeginBlocker(ctx Context) {
	if beginBlockers, ok := mgr.Config.BeginBlockers[mgr.GetHeight()]; ok {
		for _, beginBlocker := range beginBlockers {
			beginBlocker(ctx)
		}
	}
}

func (mgr *UpgradeManager) RegisterBeginBlocker(name string, beginBlocker func(Context)) {
	height := mgr.GetUpgradeHeight(name)
	if height == 0 {
		panic(fmt.Errorf("no UpgradeHeight found for %s", name))
	}

	if mgr.Config.BeginBlockers == nil {
		mgr.Config.BeginBlockers = make(map[int64][]func(ctx Context))
	}

	if beginBlockers, ok := mgr.Config.BeginBlockers[height]; ok {
		beginBlockers = append(beginBlockers, beginBlocker)
		mgr.Config.BeginBlockers[height] = beginBlockers
	} else {
		mgr.Config.BeginBlockers[height] = []func(Context){beginBlocker}
	}
}

func (mgr *UpgradeManager) AddUpgradeHeight(name string, height int64) {
	if mgr.Config.HeightMap == nil {
		mgr.Config.HeightMap = map[string]int64{}
	}

	mgr.Config.HeightMap[name] = height
}

func (mgr *UpgradeManager) GetUpgradeHeight(name string) int64 {
	if mgr.Config.HeightMap == nil {
		return 0
	}
	return mgr.Config.HeightMap[name]
}

func (mgr *UpgradeManager) RegisterStoreKeys(upgradeName string, storeKeyNames ...string) {
	height := mgr.GetUpgradeHeight(upgradeName)
	if height == 0 {
		panic(fmt.Errorf("no UpgradeHeight found for %s", upgradeName))
	}

	if mgr.Config.StoreKeyMap == nil {
		mgr.Config.StoreKeyMap = map[string]int64{}
	}

	for _, storeKeyName := range storeKeyNames {
		mgr.Config.StoreKeyMap[storeKeyName] = height
	}
}

func (mgr *UpgradeManager) RegisterMsgTypes(upgradeName string, msgTypes ...string) {
	height := mgr.GetUpgradeHeight(upgradeName)
	if height == 0 {
		panic(fmt.Errorf("no UpgradeHeight found for %s", upgradeName))
	}

	if mgr.Config.MsgTypeMap == nil {
		mgr.Config.MsgTypeMap = map[string]int64{}
	}

	for _, msgType := range msgTypes {
		mgr.Config.MsgTypeMap[msgType] = height
	}
}

func (mgr *UpgradeManager) GetStoreKeyHeight(storeKeyName string) int64 {
	if mgr.Config.StoreKeyMap == nil {
		return 0
	}

	return mgr.Config.StoreKeyMap[storeKeyName]
}

func (mgr *UpgradeManager) GetMsgTypeHeight(msgType string) int64 {
	if mgr.Config.MsgTypeMap == nil {
		return 0
	}

	return mgr.Config.MsgTypeMap[msgType]
}

func IsUpgradeHeight(name string) bool {
	upgradeHeight := UpgradeMgr.GetUpgradeHeight(name)
	if upgradeHeight == 0 {
		return false
	}

	return upgradeHeight == UpgradeMgr.GetHeight()
}

func IsUpgrade(name string) bool {
	upgradeHeight := UpgradeMgr.GetUpgradeHeight(name)
	if upgradeHeight == 0 {
		return false
	}

	return UpgradeMgr.GetHeight() >= upgradeHeight
}

func ShouldCommitStore(storeKeyName string) bool {
	storeKeyHeight := UpgradeMgr.GetStoreKeyHeight(storeKeyName)
	if storeKeyHeight == 0 {
		return true
	}

	return UpgradeMgr.GetHeight() >= storeKeyHeight
}

func ShouldSetStoreVersion(storeKeyName string) bool {
	storeKeyHeight := UpgradeMgr.GetStoreKeyHeight(storeKeyName)
	if storeKeyHeight == 0 {
		return false
	}

	return UpgradeMgr.GetHeight() == storeKeyHeight
}

func IsMsgTypeSupported(msgType string) bool {
	msgTypeHeight := UpgradeMgr.GetMsgTypeHeight(msgType)
	if msgTypeHeight == 0 {
		return true
	}

	return UpgradeMgr.GetHeight() >= msgTypeHeight
}

func Upgrade(name string, before func(), in func(), after func()) {
	// if no special logic for the UpgradeHeight, apply the `after` logic
	if in == nil {
		in = after
	}

	if IsUpgradeHeight(name) {
		if in != nil {
			in()
		}
	} else if IsUpgrade(name) {
		if after != nil {
			after()
		}
	} else {
		if before != nil {
			before()
		}
	}
}
