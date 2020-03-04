package cross_chain

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/binance-chain/node/plugins/tokens"
)

type Keeper struct {
	cdc *codec.Codec // The wire codec for binary encoding/decoding.

	storeKey sdk.StoreKey // The key used to access the store from the Context.

	TokenMapper tokens.Mapper

	BankKeeper bank.Keeper
}

func NewKeeper(cdc *codec.Codec, storeKey sdk.StoreKey, tokenMapper tokens.Mapper, bankKeeper bank.Keeper) Keeper {
	return Keeper{
		cdc:         cdc,
		TokenMapper: tokenMapper,
		BankKeeper:  bankKeeper,
		storeKey:    storeKey,
	}
}
