package app_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	appPkg "github.com/BiJie/BinanceChain/app"
	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/types"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
	dextypes "github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/wire"
)

// this file has to named with suffix _test, this is a golang bug: https://github.com/golang/go/issues/24895
var (
	keeper *orderPkg.Keeper
	buyer  sdk.AccAddress
	seller sdk.AccAddress
	am     auth.AccountMapper
	ctx    sdk.Context
	app    types.ChainApp
	cdc    *wire.Codec
)

func setup(t *testing.T) (*assert.Assertions, *require.Assertions) {
	logger := log.NewTMLogger(os.Stdout)

	db := dbm.NewMemDB()
	app = appPkg.NewBinanceChain(logger, db, os.Stdout)
	//ctx = app.NewContext(false, abci.Header{ChainID: "mychainid"})
	ctx = app.GetContextForCheckState()
	cdc = app.GetCodec()

	keeper = app.(*appPkg.BinanceChain).DexKeeper
	tradingPair := dextypes.NewTradingPair("XYZ", "BNB", 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	am = app.(*appPkg.BinanceChain).AccountKeeper
	_, buyerAcc := testutils.NewAccount(ctx, am, 100000000000) // give user enough coins to pay the fee
	buyer = buyerAcc.GetAddress()

	_, sellerAcc := testutils.NewAccount(ctx, am, 100000000000)
	seller = sellerAcc.GetAddress()

	return assert.New(t), require.New(t)
}
