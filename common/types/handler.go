package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Handler defines the core of the state transition function of an application.
type Handler func(ctx sdk.Context, msg sdk.Msg, simulate bool) sdk.Result
