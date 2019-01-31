package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	orderPkg "github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/wire"
)

var (
	keeper    *orderPkg.Keeper
	buyer     sdk.AccAddress
	buyerAcc  sdk.Account
	seller    sdk.AccAddress
	sellerAcc sdk.Account
	am        auth.AccountKeeper
	ctx       sdk.Context
	app       *BinanceChain
	cdc       *wire.Codec
)
