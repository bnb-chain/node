// nolint
package stake

import (
	"github.com/cosmos/cosmos-sdk/x/stake/keeper"
	"github.com/cosmos/cosmos-sdk/x/stake/querier"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

type (
	Keeper                     = keeper.Keeper
	Validator                  = types.Validator
	Description                = types.Description
	Commission                 = types.Commission
	Delegation                 = types.Delegation
	UnbondingDelegation        = types.UnbondingDelegation
	Redelegation               = types.Redelegation
	Params                     = types.Params
	Pool                       = types.Pool
	MsgCreateValidator         = types.MsgCreateValidator
	MsgRemoveValidator         = types.MsgRemoveValidator
	MsgCreateValidatorProposal = types.MsgCreateValidatorProposal
	MsgEditValidator           = types.MsgEditValidator
	MsgDelegate                = types.MsgDelegate
	MsgBeginUnbonding          = types.MsgBeginUnbonding
	MsgBeginRedelegate         = types.MsgBeginRedelegate
	GenesisState               = types.GenesisState
	QueryDelegatorParams       = querier.QueryDelegatorParams
	QueryValidatorParams       = querier.QueryValidatorParams
	QueryBondsParams           = querier.QueryBondsParams
	CreateValidatorJsonMsg     = types.CreateValidatorJsonMsg
	QueryTopValidatorsParams   = querier.QueryTopValidatorsParams
	BaseParams                 = querier.BaseParams

	MsgCreateSideChainValidator = types.MsgCreateSideChainValidator
	MsgEditSideChainValidator   = types.MsgEditSideChainValidator
	MsgSideChainDelegate        = types.MsgSideChainDelegate
	MsgSideChainRedelegate      = types.MsgSideChainRedelegate
	MsgSideChainUndelegate      = types.MsgSideChainUndelegate

	SideDistributionEvent      = types.SideDistributionEvent
	DistributionData           = types.DistributionData
	SideCompletedUBDEvent      = types.SideCompletedUBDEvent
	SideCompletedREDEvent      = types.SideCompletedREDEvent
	ValidatorUpdateEvent       = types.ValidatorUpdateEvent
	ValidatorRemovedEvent      = types.ValidatorRemovedEvent
	DelegationUpdateEvent      = types.DelegationUpdateEvent
	DelegationRemovedEvent     = types.DelegationRemovedEvent
	SideDelegationRemovedEvent = types.SideDelegationRemovedEvent
	SideDelegationUpdateEvent  = types.SideDelegationUpdateEvent
	UBDUpdateEvent             = types.UBDUpdateEvent
	SideUBDUpdateEvent         = types.SideUBDUpdateEvent
	REDUpdateEvent             = types.REDUpdateEvent
	SideREDUpdateEvent         = types.SideREDUpdateEvent
	DVVTriplet                 = types.DVVTriplet
	DVPair                     = types.DVPair
	DelegateEvent              = types.DelegateEvent
	UndelegateEvent            = types.UndelegateEvent
	RedelegateEvent            = types.RedelegateEvent
	SideDelegateEvent          = types.SideDelegateEvent
	SideUndelegateEvent        = types.SideUnDelegateEvent
	SideRedelegateEvent        = types.SideRedelegateEvent
	SideElectedValidatorsEvent = types.SideElectedValidatorsEvent
)

var (
	NewKeeper = keeper.NewKeeper

	GetValidatorKey              = keeper.GetValidatorKey
	GetValidatorByConsAddrKey    = keeper.GetValidatorByConsAddrKey
	GetValidatorsByPowerIndexKey = keeper.GetValidatorsByPowerIndexKey
	GetDelegationKey             = keeper.GetDelegationKey
	GetDelegationsKey            = keeper.GetDelegationsKey
	PoolKey                      = keeper.PoolKey
	IntraTxCounterKey            = keeper.IntraTxCounterKey
	LastValidatorPowerKey        = keeper.LastValidatorPowerKey
	LastTotalPowerKey            = keeper.LastTotalPowerKey
	ValidatorsKey                = keeper.ValidatorsKey
	ValidatorsByConsAddrKey      = keeper.ValidatorsByConsAddrKey
	ValidatorsByPowerIndexKey    = keeper.ValidatorsByPowerIndexKey
	DelegationKey                = keeper.DelegationKey
	GetUBDKey                    = keeper.GetUBDKey
	GetUBDByValIndexKey          = keeper.GetUBDByValIndexKey
	GetUBDsKey                   = keeper.GetUBDsKey
	GetUBDsByValIndexKey         = keeper.GetUBDsByValIndexKey
	GetREDKey                    = keeper.GetREDKey
	GetREDByValSrcIndexKey       = keeper.GetREDByValSrcIndexKey
	GetREDByValDstIndexKey       = keeper.GetREDByValDstIndexKey
	GetREDsKey                   = keeper.GetREDsKey
	GetREDsFromValSrcIndexKey    = keeper.GetREDsFromValSrcIndexKey
	GetREDsToValDstIndexKey      = keeper.GetREDsToValDstIndexKey
	GetREDsByDelToValDstIndexKey = keeper.GetREDsByDelToValDstIndexKey
	TestingUpdateValidator       = keeper.TestingUpdateValidator
	MigratePowerRankKey          = keeper.MigratePowerRankKey

	DefaultParamspace = keeper.DefaultParamspace
	KeyUnbondingTime  = types.KeyUnbondingTime
	KeyMaxValidators  = types.KeyMaxValidators
	KeyBondDenom      = types.KeyBondDenom

	DefaultParams           = types.DefaultParams
	InitialPool             = types.InitialPool
	NewValidator            = types.NewValidator
	NewValidatorWithFeeAddr = types.NewValidatorWithFeeAddr
	NewSideChainValidator   = types.NewSideChainValidator
	NewDescription          = types.NewDescription
	NewCommission           = types.NewCommission
	NewCommissionMsg        = types.NewCommissionMsg
	NewCommissionWithTime   = types.NewCommissionWithTime
	NewGenesisState         = types.NewGenesisState
	DefaultGenesisState     = types.DefaultGenesisState
	RegisterCodec           = types.RegisterCodec

	NewMsgCreateValidator           = types.NewMsgCreateValidator
	NewMsgRemoveValidator           = types.NewMsgRemoveValidator
	NewMsgCreateValidatorOnBehalfOf = types.NewMsgCreateValidatorOnBehalfOf
	NewMsgEditValidator             = types.NewMsgEditValidator
	NewMsgDelegate                  = types.NewMsgDelegate
	NewMsgBeginUnbonding            = types.NewMsgBeginUnbonding
	NewMsgBeginRedelegate           = types.NewMsgBeginRedelegate

	NewMsgCreateSideChainValidator           = types.NewMsgCreateSideChainValidator
	NewMsgCreateSideChainValidatorOnBehalfOf = types.NewMsgCreateSideChainValidatorOnBehalfOf
	NewMsgEditSideChainValidator             = types.NewMsgEditSideChainValidator
	NewMsgSideChainDelegate                  = types.NewMsgSideChainDelegate
	NewMsgSideChainRedelegate                = types.NewMsgSideChainRedelegate
	NewMsgSideChainUndelegate                = types.NewMsgSideChainUndelegate

	NewQuerier    = querier.NewQuerier
	NewBaseParams = querier.NewBaseParams

	DelegationAccAddr = keeper.DelegationAccAddr
)

const (
	QueryValidators                    = querier.QueryValidators
	QueryValidator                     = querier.QueryValidator
	QueryValidatorUnbondingDelegations = querier.QueryValidatorUnbondingDelegations
	QueryValidatorRedelegations        = querier.QueryValidatorRedelegations
	QueryDelegation                    = querier.QueryDelegation
	QueryUnbondingDelegation           = querier.QueryUnbondingDelegation
	QueryDelegatorDelegations          = querier.QueryDelegatorDelegations
	QueryDelegatorUnbondingDelegations = querier.QueryDelegatorUnbondingDelegations
	QueryDelegatorRedelegations        = querier.QueryDelegatorRedelegations
	QueryDelegatorValidators           = querier.QueryDelegatorValidators
	QueryDelegatorValidator            = querier.QueryDelegatorValidator
	QueryPool                          = querier.QueryPool
	QueryParameters                    = querier.QueryParameters

	Topic = types.Topic
)

const (
	DefaultCodespace      = types.DefaultCodespace
	CodeInvalidValidator  = types.CodeInvalidValidator
	CodeInvalidDelegation = types.CodeInvalidDelegation
	CodeInvalidInput      = types.CodeInvalidInput
	CodeValidatorJailed   = types.CodeValidatorJailed
	CodeUnauthorized      = types.CodeUnauthorized
	CodeInternal          = types.CodeInternal
	CodeUnknownRequest    = types.CodeUnknownRequest
)

var (
	ErrNilValidatorAddr           = types.ErrNilValidatorAddr
	ErrNoValidatorFound           = types.ErrNoValidatorFound
	ErrValidatorOwnerExists       = types.ErrValidatorOwnerExists
	ErrValidatorPubKeyExists      = types.ErrValidatorPubKeyExists
	ErrValidatorSideConsAddrExist = types.ErrValidatorSideConsAddrExists
	ErrInvalidDelegator           = types.ErrInvalidDelegator
	ErrValidatorJailed            = types.ErrValidatorJailed
	ErrInvalidProposal            = types.ErrInvalidProposal
	ErrBadRemoveValidator         = types.ErrBadRemoveValidator
	ErrDescriptionLength          = types.ErrDescriptionLength
	ErrCommissionNegative         = types.ErrCommissionNegative
	ErrCommissionHuge             = types.ErrCommissionHuge

	ErrNilDelegatorAddr          = types.ErrNilDelegatorAddr
	ErrBadDenom                  = types.ErrBadDenom
	ErrBadDelegationAmount       = types.ErrBadDelegationAmount
	ErrNoDelegation              = types.ErrNoDelegation
	ErrBadDelegatorAddr          = types.ErrBadDelegatorAddr
	ErrNoDelegatorForAddress     = types.ErrNoDelegatorForAddress
	ErrInsufficientShares        = types.ErrInsufficientShares
	ErrDelegationValidatorEmpty  = types.ErrDelegationValidatorEmpty
	ErrNotEnoughDelegationShares = types.ErrNotEnoughDelegationShares
	ErrBadSharesAmount           = types.ErrBadSharesAmount
	ErrBadSharesPercent          = types.ErrBadSharesPercent

	ErrInvalidSideChainId = types.ErrInvalidSideChainId

	ErrNotMature             = types.ErrNotMature
	ErrNoUnbondingDelegation = types.ErrNoUnbondingDelegation
	ErrNoRedelegation        = types.ErrNoRedelegation
	ErrBadRedelegationSrc    = types.ErrBadRedelegationSrc
	ErrBadRedelegationDst    = types.ErrBadRedelegationDst
	ErrSelfRedelegation      = types.ErrSelfRedelegation
	ErrInvalidRedelegator    = types.ErrInvalidRedelegator

	ErrBothShareMsgsGiven    = types.ErrBothShareMsgsGiven
	ErrNeitherShareMsgsGiven = types.ErrNeitherShareMsgsGiven
	ErrMissingSignature      = types.ErrMissingSignature
)
