package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"
)

var (
	// bnb prefix address:  bnb1v8vkkymvhe2sf7gd2092ujc6hweta38xadu2pj
	// tbnb prefix address: tbnb1v8vkkymvhe2sf7gd2092ujc6hweta38xnc4wpr
	PegAccount = sdk.AccAddress(crypto.AddressHash([]byte("BinanceChainPegAccount")))
)
