package tx

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/types"
)

type contextKey int

const (
	contextKeyFee contextKey = iota
)

// add the signers to the context
func WithFee(ctx types.Context, fee types.Fee) types.Context {
	return ctx.WithValue(contextKeyFee, fee)
}

// get the signers from the context
func Fee(ctx types.Context) types.Fee {
	v := ctx.Value(contextKeyFee)
	if v == nil {
		return types.Fee{}
	}
	return v.(types.Fee)
}
