package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// RelayPackagesChannelId is not a communication channel actually, we just use it to record sequence.
	RelayPackagesChannelName               = "relayPackages"
	RelayPackagesChannelId   sdk.ChannelID = 0x00
)

func GetClaimId(chainId sdk.ChainID, channelId sdk.ChannelID, sequence uint64) string {
	return fmt.Sprintf("%d:%d:%d", chainId, channelId, sequence)
}

// Claim contains an arbitrary claim with arbitrary content made by a given validator
type Claim struct {
	ID               string         `json:"id"`
	ValidatorAddress sdk.ValAddress `json:"validator_address"`
	Payload          string         `json:"payload"`
}

// NewClaim returns a new Claim
func NewClaim(id string, validatorAddress sdk.ValAddress, payload string) Claim {
	return Claim{
		ID:               id,
		ValidatorAddress: validatorAddress,
		Payload:          payload,
	}
}
