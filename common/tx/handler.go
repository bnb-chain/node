package tx

import sdk "github.com/cosmos/cosmos-sdk/types"

// Handler defines the core of the state transition function of an application.
type Handler func(ctx sdk.Context, msg Msg) sdk.Result

// AnteHandler authenticates transactions, before their internal messages are handled.
// If newCtx.IsZero(), ctx is used instead.
type AnteHandler func(ctx sdk.Context, tx Tx) (newCtx sdk.Context, result sdk.Result, abort bool)
