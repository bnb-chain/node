package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type OracleKeeper interface {
	GetClaimTypeName(claimType sdk.ClaimType) string
	GetCurrentSequence(ctx sdk.Context, claimType sdk.ClaimType) int64
	IncreaseSequence(ctx sdk.Context, claimType sdk.ClaimType) int64
	RegisterClaimType(claimType sdk.ClaimType, claimTypeName string, hooks sdk.ClaimHooks) error
}
