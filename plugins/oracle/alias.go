package oracle

// nolint
// autogenerated code using github.com/rigelrozanski/multitool
// aliases generated for the following subdirectories:
// ALIASGEN: github.com/cosmos/peggy/x/oracle/keeper
// ALIASGEN: github.com/cosmos/peggy/x/oracle/types

import (
	"github.com/binance-chain/node/plugins/oracle/keeper"
	"github.com/binance-chain/node/plugins/oracle/types"
)

const (
	DefaultConsensusNeeded = types.DefaultConsensusNeeded
	PendingStatusText      = types.PendingStatusText
	SuccessStatusText      = types.SuccessStatusText
	FailedStatusText       = types.FailedStatusText
)

var (
	// functions aliases
	NewKeeper = keeper.NewKeeper

	NewClaim                         = types.NewClaim
	ErrProphecyNotFound              = types.ErrProphecyNotFound
	ErrMinimumConsensusNeededInvalid = types.ErrMinimumConsensusNeededInvalid
	ErrNoClaims                      = types.ErrNoClaims
	ErrInvalidIdentifier             = types.ErrInvalidIdentifier
	ErrProphecyFinalized             = types.ErrProphecyFinalized
	ErrDuplicateMessage              = types.ErrDuplicateMessage
	ErrInvalidClaim                  = types.ErrInvalidClaim
	ErrInvalidValidator              = types.ErrInvalidValidator
	ErrInternalDB                    = types.ErrInternalDB
	NewProphecy                      = types.NewProphecy
	NewStatus                        = types.NewStatus

	// variable aliases

	StatusTextToString = types.StatusTextToString
	StringToStatusText = types.StringToStatusText
)

type (
	Keeper     = keeper.Keeper
	Claim      = types.Claim
	Prophecy   = types.Prophecy
	DBProphecy = types.DBProphecy
	Status     = types.Status
	StatusText = types.StatusText
)
