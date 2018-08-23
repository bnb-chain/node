package tx

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/types"
)

type contextKey int // local to the auth module

const (
	contextKeyFee contextKey = iota
)

// add the signers to the context
func WithFee(ctx sdk.Context, fee types.Fee) sdk.Context {
	return ctx.WithValue(contextKeyFee, fee)
}

// get the signers from the context
func Fee(ctx sdk.Context) types.Fee {
	v := ctx.Value(contextKeyFee)
	if v == nil {
		return types.Fee{}
	}
	return v.(types.Fee)
}
