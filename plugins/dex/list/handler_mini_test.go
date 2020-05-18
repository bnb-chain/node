package list

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	tokenStore "github.com/binance-chain/node/plugins/tokens/store"
)

func setChainVersion() {
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP8, -1)
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP70, -1)
}

func resetChainVersion() {
	upgrade.Mgr.Config.HeightMap = nil
}

func setupForMini(ctx sdk.Context, tokenMapper tokenStore.Mapper, t *testing.T ){
	err := tokenMapper.NewToken(ctx, &types.Token{
		Name:        "Bitcoin",
		Symbol:      "BTC-000",
		OrigSymbol:  "BTC",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = tokenMapper.NewToken(ctx, &types.Token{
		Name:        "Native Token",
		Symbol:      types.NativeTokenSymbol,
		OrigSymbol:  types.NativeTokenSymbol,
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = tokenMapper.NewToken(ctx, &types.Token{
		Name:        "Bitcoin Mini",
		Symbol:      "BTC-000M",
		OrigSymbol:  "BTC",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

}

func TestHandleListMiniIdenticalSymbols(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	setupForMini(ctx, tokenMapper, t)
	result := handleListMini(ctx, orderKeeper, tokenMapper, ListMiniMsg{
		From: sdk.AccAddress("testacc"),
		BaseAssetSymbol: "BTC-000M",
		QuoteAssetSymbol: "BTC-000M",
		InitPrice: 1000,
	})
	require.Contains(t, result.Log, "base asset symbol should not be identical to quote asset symbol")
}

func TestHandleListMiniWrongBaseSymbol(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	setupForMini(ctx, tokenMapper, t)
	result := handleListMini(ctx, orderKeeper, tokenMapper, ListMiniMsg{
		From: sdk.AccAddress("testacc"),
		BaseAssetSymbol: "BTC",
		QuoteAssetSymbol: "BNB",
		InitPrice: 1000,
	})
	//require.Equal(t, result.Code, sdk.ABCICodeOK)
	require.Contains(t, result.Log, "token(BTC) not found")
}

func TestHandleListMiniRight(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	setupForMini(ctx, tokenMapper, t)
	result := handleListMini(ctx, orderKeeper, tokenMapper, ListMiniMsg{
		From: sdk.AccAddress("testacc"),
		BaseAssetSymbol: "BTC-000M",
		QuoteAssetSymbol: "BNB",
		InitPrice: 1000,
	})
	require.Equal(t, result.Code, sdk.ABCICodeOK)
}
