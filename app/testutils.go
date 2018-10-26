package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/wire"
)

var (
	keeper    *orderPkg.Keeper
	buyer     sdk.AccAddress
	buyerAcc  auth.Account
	seller    sdk.AccAddress
	sellerAcc auth.Account
	am        auth.AccountKeeper
	ctx       sdk.Context
	app       *BinanceChain
	cdc       *wire.Codec
)
