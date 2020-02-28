package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// Claim contains an arbitrary claim with arbitrary content made by a given validator
type Claim struct {
	ID               string         `json:"id"`
	ValidatorAddress sdk.AccAddress `json:"validator_address"`
	Content          string         `json:"content"`
}

// NewClaim returns a new Claim
func NewClaim(id string, validatorAddress sdk.AccAddress, content string) Claim {
	return Claim{
		ID:               id,
		ValidatorAddress: validatorAddress,
		Content:          content,
	}
}
