package ibc

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
)

func createTestInput(t *testing.T, isCheckTx bool) (sdk.Context, Keeper) {
	keyIBC := sdk.NewKVStoreKey("ibc")
	keySideChain := sdk.NewKVStoreKey("sc")
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyIBC, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySideChain, sdk.StoreTypeIAVL, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	mode := sdk.RunTxModeDeliver
	if isCheckTx {
		mode = sdk.RunTxModeCheck
	}
	keyParams := sdk.NewKVStoreKey("params")
	tkeyParams := sdk.NewTransientStoreKey("transient_params")

	cdc := createTestCodec()
	pk := params.NewKeeper(cdc, keyParams, tkeyParams)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "foochainid"}, mode, log.NewNopLogger())
	scKeeper := sidechain.NewKeeper(keySideChain, pk.Subspace(sidechain.DefaultParamspace), cdc)
	ibcKeeper := NewKeeper(keyIBC, pk.Subspace(DefaultParamspace), DefaultCodespace, scKeeper)

	return ctx, ibcKeeper
}

func TestKeeper(t *testing.T) {
	sourceChainID := sdk.ChainID(0x0001)

	destChainName := "bsc"
	destChainID := sdk.ChainID(0x000f)

	channelName := "transfer"
	channelID := sdk.ChannelID(0x01)

	ctx, keeper := createTestInput(t, true)

	keeper.sideKeeper.SetChannelSendPermission(ctx, destChainID, channelID, sdk.ChannelAllow)

	keeper.sideKeeper.SetSrcChainID(sourceChainID)
	require.NoError(t, keeper.sideKeeper.RegisterDestChain(destChainName, destChainID))
	require.NoError(t, keeper.sideKeeper.RegisterChannel(channelName, channelID, nil))
	testSynFee := big.NewInt(100)

	value := []byte{0x00}
	sequence, err := keeper.CreateRawIBCPackage(ctx, destChainName, channelName, sdk.SynCrossChainPackageType, value, *testSynFee)
	require.NoError(t, err)
	require.Equal(t, uint64(0), sequence)

	value = []byte{0x00, 0x01}
	sequence, err = keeper.CreateRawIBCPackage(ctx, destChainName, channelName, sdk.SynCrossChainPackageType, value, *testSynFee)
	require.NoError(t, err)
	require.Equal(t, uint64(1), sequence)
	value = []byte{0x00, 0x01, 0x02}
	sequence, err = keeper.CreateRawIBCPackage(ctx, destChainName, channelName, sdk.SynCrossChainPackageType, value, *testSynFee)
	require.NoError(t, err)
	require.Equal(t, uint64(2), sequence)
	value = []byte{0x00, 0x01, 0x02, 0x03}
	sequence, err = keeper.CreateRawIBCPackage(ctx, destChainName, channelName, sdk.SynCrossChainPackageType, value, *testSynFee)
	require.NoError(t, err)
	require.Equal(t, uint64(3), sequence)
	value = []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	sequence, err = keeper.CreateRawIBCPackage(ctx, destChainName, channelName, sdk.SynCrossChainPackageType, value, *testSynFee)
	require.NoError(t, err)
	require.Equal(t, uint64(4), sequence)

	keeper.CleanupIBCPackage(ctx, destChainName, channelName, 3)

	ibcPackage, sdkErr := keeper.GetIBCPackage(ctx, destChainName, channelName, 0)
	require.NoError(t, sdkErr)
	require.Nil(t, ibcPackage)
	ibcPackage, sdkErr = keeper.GetIBCPackage(ctx, destChainName, channelName, 1)
	require.NoError(t, sdkErr)
	require.Nil(t, ibcPackage)
	ibcPackage, sdkErr = keeper.GetIBCPackage(ctx, destChainName, channelName, 2)
	require.NoError(t, sdkErr)
	require.Nil(t, ibcPackage)
	ibcPackage, sdkErr = keeper.GetIBCPackage(ctx, destChainName, channelName, 3)
	require.NoError(t, sdkErr)
	require.Nil(t, ibcPackage)
	ibcPackage, sdkErr = keeper.GetIBCPackage(ctx, destChainName, channelName, 4)
	require.NoError(t, sdkErr)
	require.NotNil(t, ibcPackage)

	require.NoError(t, keeper.sideKeeper.RegisterDestChain("btc", sdk.ChainID(0x0002)))
	keeper.sideKeeper.SetChannelSendPermission(ctx, sdk.ChainID(0x0002), channelID, sdk.ChannelAllow)

	sequence, err = keeper.CreateRawIBCPackage(ctx, "btc", channelName, sdk.SynCrossChainPackageType, value, *testSynFee)
	require.NoError(t, err)
	require.Equal(t, uint64(0), sequence)

	require.NoError(t, keeper.sideKeeper.RegisterChannel("mockChannel", sdk.ChannelID(2), nil))
	keeper.sideKeeper.SetChannelSendPermission(ctx, destChainID, sdk.ChannelID(2), sdk.ChannelAllow)
	sequence, err = keeper.CreateRawIBCPackage(ctx, destChainName, "mockChannel", sdk.SynCrossChainPackageType, value, *testSynFee)
	require.NoError(t, err)
	require.Equal(t, uint64(0), sequence)
	require.Equal(t, uint64(0), sequence)

}

func createTestCodec() *codec.Codec {
	cdc := codec.New()
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	return cdc
}
