package order

import sdk "github.com/cosmos/cosmos-sdk/types"

type contextKey int

const (
	contextKeySettlement contextKey = iota
)

func WithSettlement(ctx sdk.Context, transMap []Transfer) sdk.Context {
	return ctx.WithValue(contextKeySettlement, transMap)
}

func Settlement(ctx sdk.Context) []Transfer {
	v := ctx.Value(contextKeySettlement)
	if v == nil {
		return []Transfer{}
	}
	return v.([]Transfer)
}
