package rest

import (
	"github.com/gorilla/mux"
	cmn "github.com/tendermint/tendermint/libs/common"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/BiJie/BinanceChain/wire"
)

// https://github.com/tendermint/tendermint/blob/05a76fb517f50da27b4bfcdc7b4cf185fc61eff6/crypto/crypto.go#L14
type Address = cmn.HexBytes

var msgCdc = wire.NewCodec()

// RegisterRoutes registers staking-related REST handlers to a router
func RegisterRoutes(
	ctx context.CoreContext,
	r *mux.Router,
	cdc *wire.Codec,
) {
	registerBooksRoute(ctx, r, cdc)
}
