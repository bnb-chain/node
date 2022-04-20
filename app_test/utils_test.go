package app_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	appPkg "github.com/bnb-chain/node/app"
	"github.com/bnb-chain/node/common/testutils"
	"github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/common/upgrade"
	orderPkg "github.com/bnb-chain/node/plugins/dex/order"
	dextypes "github.com/bnb-chain/node/plugins/dex/types"
	"github.com/bnb-chain/node/wire"
)

// this file has to named with suffix _test, this is a golang bug: https://github.com/golang/go/issues/24895
var (
	keeper *orderPkg.DexKeeper
	buyer  sdk.AccAddress
	seller sdk.AccAddress
	am     auth.AccountKeeper
	ctx    sdk.Context
	app    types.ChainApp
	cdc    *wire.Codec
)

func setup(t *testing.T, symbol string, upgrade bool) (ass *assert.Assertions, req *require.Assertions, pair string) {
	baseAssetSymbol := symbol
	logger := log.NewTMLogger(os.Stdout)

	db := dbm.NewMemDB()
	app = appPkg.NewBinanceChain(logger, db, os.Stdout)
	//ctx = app.NewContext(false, abci.Header{ChainID: "mychainid"})
	ctx = app.GetContextForCheckState()
	cdc = app.GetCodec()

	if upgrade {
		setChainVersion()
	}

	keeper = app.(*appPkg.BinanceChain).DexKeeper
	tradingPair := dextypes.NewTradingPair(baseAssetSymbol, types.NativeTokenSymbol, 1e8)
	keeper.PairMapper.AddTradingPair(ctx, tradingPair)
	keeper.AddEngine(tradingPair)

	am = app.(*appPkg.BinanceChain).AccountKeeper
	_, buyerAcc := testutils.NewAccountForPub(ctx, am, 100000000000, 100000000000, 100000000000, symbol) // give user enough coins to pay the fee
	buyer = buyerAcc.GetAddress()

	_, sellerAcc := testutils.NewAccountForPub(ctx, am, 100000000000, 100000000000, 100000000000, symbol)
	seller = sellerAcc.GetAddress()

	pair = fmt.Sprintf("%s_%s", baseAssetSymbol, types.NativeTokenSymbol)

	return assert.New(t), require.New(t), pair
}

func setChainVersion() {
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP8, -1)
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP70, -1)
}

func resetChainVersion() {
	upgrade.Mgr.Config.HeightMap = nil
}
