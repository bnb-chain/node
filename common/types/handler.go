package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// Handler defines the core of the state transition function of an application.
type Handler func(ctx Context, msg sdk.Msg) sdk.Result

// AnteHandler authenticates transactions, before their internal messages are handled.
// If newCtx.IsZero(), ctx is used instead.
type AnteHandler func(ctx Context, tx sdk.Tx) (newCtx Context, result sdk.Result, abort bool)
