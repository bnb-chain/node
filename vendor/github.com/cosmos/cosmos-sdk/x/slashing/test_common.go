package slashing

import (
	"encoding/hex"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

// TODO remove dependencies on staking (should only refer to validator set type from sdk)

var (
	pks = []crypto.PubKey{
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB50"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB51"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB52"),
	}
	addrs = []sdk.ValAddress{
		sdk.ValAddress(pks[0].Address()),
		sdk.ValAddress(pks[1].Address()),
		sdk.ValAddress(pks[2].Address()),
	}
	initCoins = sdk.NewDecWithoutFra(20000).RawInt()
)

func createTestCodec() *codec.Codec {
	cdc := codec.New()
	sdk.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	bank.RegisterCodec(cdc)
	stake.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	return cdc
}

func getAccountCache(cdc *codec.Codec, ms sdk.MultiStore, accountKey *sdk.KVStoreKey) sdk.AccountCache {
	accountStore := ms.GetKVStore(accountKey)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	return auth.NewAccountCache(accountStoreCache)
}

func createTestInput(t *testing.T, defaults Params) (sdk.Context, bank.Keeper, stake.Keeper, params.Subspace, Keeper) {
	keyAcc := sdk.NewKVStoreKey("acc")
	keyStake := sdk.NewKVStoreKey("stake")
	keyStakeReward := sdk.NewKVStoreKey("stake_reward")
	tkeyStake := sdk.NewTransientStoreKey("transient_stake")
	keySlashing := sdk.NewKVStoreKey("slashing")
	keyParams := sdk.NewKVStoreKey("params")
	tkeyParams := sdk.NewTransientStoreKey("transient_params")
	keyIbc := sdk.NewKVStoreKey("ibc")
	keySideChain := sdk.NewKVStoreKey("sc")

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyStake, sdk.StoreTypeTransient, nil)
	ms.MountStoreWithDB(keyStake, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStakeReward, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySlashing, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyIbc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySideChain, sdk.StoreTypeIAVL, db)

	err := ms.LoadLatestVersion()
	require.Nil(t, err)
	ctx := sdk.NewContext(ms, abci.Header{Time: time.Unix(0, 0)}, sdk.RunTxModeDeliver, log.NewTMLogger(os.Stdout))
	cdc := createTestCodec()
	accountKeeper := auth.NewAccountKeeper(cdc, keyAcc, auth.ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, keyAcc)
	ctx = ctx.WithAccountCache(accountCache)

	ck := bank.NewBaseKeeper(accountKeeper)
	paramsKeeper := params.NewKeeper(cdc, keyParams, tkeyParams)
	scKeeper := sidechain.NewKeeper(keySideChain, paramsKeeper.Subspace(sidechain.DefaultParamspace), cdc)
	ibcKeeper := ibc.NewKeeper(keyIbc, paramsKeeper.Subspace(ibc.DefaultParamspace), ibc.DefaultCodespace, scKeeper)
	sk := stake.NewKeeper(cdc, keyStake, keyStakeReward, tkeyStake, ck, nil, paramsKeeper.Subspace(stake.DefaultParamspace), stake.DefaultCodespace)
	sk.SetupForSideChain(&scKeeper, &ibcKeeper)
	genesis := stake.DefaultGenesisState()

	genesis.Pool.LooseTokens = sdk.NewDec(initCoins * (int64(len(addrs))))

	_, err = stake.InitGenesis(ctx, sk, genesis)
	require.Nil(t, err)

	for _, addr := range addrs {
		_, _, err = ck.AddCoins(ctx, sdk.AccAddress(addr), sdk.Coins{
			{sk.GetParams(ctx).BondDenom, initCoins},
		})
	}
	require.Nil(t, err)
	paramstore := paramsKeeper.Subspace(DefaultParamspace)
	keeper := NewKeeper(cdc, keySlashing, sk, paramstore, DefaultCodespace, ck)
	sk = sk.WithHooks(keeper.Hooks())

	require.NotPanics(t, func() {
		InitGenesis(ctx, keeper, GenesisState{defaults}, genesis)
	})

	return ctx, ck, sk, paramstore, keeper
}

func createSideTestInput(t *testing.T, defaults Params) (sdk.Context, sdk.Context, bank.Keeper, stake.Keeper, params.Subspace, Keeper) {
	keyAcc := sdk.NewKVStoreKey("acc")
	keyStake := sdk.NewKVStoreKey("stake")
	keyStakeReward := sdk.NewKVStoreKey("stake_reward")
	tkeyStake := sdk.NewTransientStoreKey("transient_stake")
	keySlashing := sdk.NewKVStoreKey("slashing")
	keyParams := sdk.NewKVStoreKey("params")
	tkeyParams := sdk.NewTransientStoreKey("transient_params")
	keyIbc := sdk.NewKVStoreKey("ibc")
	keySideChain := sdk.NewKVStoreKey("sc")

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyStake, sdk.StoreTypeTransient, nil)
	ms.MountStoreWithDB(keyStake, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStakeReward, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySlashing, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyIbc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySideChain, sdk.StoreTypeIAVL, db)

	err := ms.LoadLatestVersion()
	require.Nil(t, err)
	ctx := sdk.NewContext(ms, abci.Header{Time: time.Now()}, sdk.RunTxModeDeliver, log.NewTMLogger(os.Stdout))
	cdc := createTestCodec()
	accountKeeper := auth.NewAccountKeeper(cdc, keyAcc, auth.ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, keyAcc)
	ctx = ctx.WithAccountCache(accountCache)
	ck := bank.NewBaseKeeper(accountKeeper)

	paramsKeeper := params.NewKeeper(cdc, keyParams, tkeyParams)

	scKeeper := sidechain.NewKeeper(keySideChain, paramsKeeper.Subspace(sidechain.DefaultParamspace), cdc)
	bscStorePrefix := []byte{0x99}
	scKeeper.SetSideChainIdAndStorePrefix(ctx, "bsc", bscStorePrefix)
	scKeeper.SetParams(ctx, sidechain.DefaultParams())

	ibcKeeper := ibc.NewKeeper(keyIbc, paramsKeeper.Subspace(ibc.DefaultParamspace), ibc.DefaultCodespace, scKeeper)
	// set up IBC chainID for BBC
	scKeeper.SetSrcChainID(sdk.ChainID(1))
	err = scKeeper.RegisterDestChain("bsc", sdk.ChainID(1))
	require.Nil(t, err)
	storePrefix := scKeeper.GetSideChainStorePrefix(ctx, "bsc")
	ibcKeeper.SetParams(ctx.WithSideChainKeyPrefix(storePrefix), ibc.Params{RelayerFee: ibc.DefaultRelayerFeeParam})

	sk := stake.NewKeeper(cdc, keyStake, keyStakeReward, tkeyStake, ck, nil, paramsKeeper.Subspace(stake.DefaultParamspace), stake.DefaultCodespace)
	sk.SetupForSideChain(&scKeeper, &ibcKeeper)
	genesis := stake.DefaultGenesisState()
	sideCtx := ctx.WithSideChainKeyPrefix(bscStorePrefix)
	sk.SetParams(sideCtx, stake.DefaultParams())
	sk.SetPool(sideCtx, stake.Pool{
		LooseTokens: sdk.NewDec(5e15),
	})

	genesis.Pool.LooseTokens = sdk.NewDec(initCoins * (int64(len(addrs))))

	_, err = stake.InitGenesis(ctx, sk, genesis)
	require.Nil(t, err)

	for _, addr := range addrs {
		_, _, err = ck.AddCoins(ctx, sdk.AccAddress(addr), sdk.Coins{
			{sk.GetParams(ctx).BondDenom, initCoins},
		})
	}
	require.Nil(t, err)
	paramstore := paramsKeeper.Subspace(DefaultParamspace)
	keeper := NewKeeper(cdc, keySlashing, sk, paramstore, DefaultCodespace, ck)
	sk = sk.WithHooks(keeper.Hooks())
	keeper.SetSideChain(&scKeeper)
	keeper.SetParams(sideCtx, defaults)
	scKeeper.SetChannelSendPermission(ctx, sdk.ChainID(1), sdk.ChannelID(8), sdk.ChannelAllow)

	require.NotPanics(t, func() {
		InitGenesis(ctx, keeper, GenesisState{defaults}, genesis)
	})

	sdk.UpgradeMgr.Height = 1
	sdk.UpgradeMgr.AddUpgradeHeight(sdk.LaunchBscUpgrade, 1)

	return ctx, sideCtx, ck, sk, paramstore, keeper
}

func newPubKey(pk string) (res crypto.PubKey) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		panic(err)
	}
	var pkEd ed25519.PubKeyEd25519
	copy(pkEd[:], pkBytes[:])
	return pkEd
}

func testAddr(addr string) sdk.AccAddress {
	res := []byte(addr)
	return res
}

func NewTestMsgCreateValidator(address sdk.ValAddress, pubKey crypto.PubKey, amt int64) stake.MsgCreateValidator {
	commission := stake.NewCommissionMsg(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec())
	return stake.MsgCreateValidator{
		Description:   stake.Description{},
		Commission:    commission,
		DelegatorAddr: sdk.AccAddress(address),
		ValidatorAddr: address,
		PubKey:        pubKey,
		Delegation:    sdk.NewCoin("steak", amt),
	}
}

func newTestMsgDelegate(delAddr sdk.AccAddress, valAddr sdk.ValAddress, delAmount int64) stake.MsgDelegate {
	return stake.MsgDelegate{
		DelegatorAddr: delAddr,
		ValidatorAddr: valAddr,
		Delegation:    sdk.NewCoin("steak", delAmount),
	}
}

func newTestMsgCreateSideValidator(address sdk.ValAddress, sideConsAddr, sideFeeAddr []byte, amt int64) stake.MsgCreateSideChainValidator {
	commission := stake.NewCommissionMsg(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec())
	return stake.MsgCreateSideChainValidator{
		Description:   stake.Description{},
		Commission:    commission,
		DelegatorAddr: sdk.AccAddress(address),
		ValidatorAddr: address,
		Delegation:    sdk.NewCoin("steak", amt),
		SideChainId:   "bsc",
		SideConsAddr:  sideConsAddr,
		SideFeeAddr:   sideFeeAddr,
	}
}

func newTestMsgSideUnDelegate(delAddr sdk.AccAddress, valAddr sdk.ValAddress, amount int64) stake.MsgSideChainUndelegate {
	return stake.MsgSideChainUndelegate{
		DelegatorAddr: delAddr,
		ValidatorAddr: valAddr,
		Amount:        sdk.NewCoin("steak", amount),
		SideChainId:   "bsc",
	}
}

func createSideAddr(length int) []byte {
	bz := make([]byte, length)
	_, _ = rand.Read(bz)
	return bz
}
