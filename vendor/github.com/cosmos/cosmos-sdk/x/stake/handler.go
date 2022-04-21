package stake

import (
	"bytes"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake/keeper"
	"github.com/cosmos/cosmos-sdk/x/stake/tags"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

func NewHandler(k keeper.Keeper, govKeeper gov.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		// NOTE msg already has validate basic run
		switch msg := msg.(type) {
		case types.MsgCreateValidatorProposal:
			return handleMsgCreateValidatorAfterProposal(ctx, msg, k, govKeeper)
		case types.MsgRemoveValidator:
			return handleMsgRemoveValidatorAfterProposal(ctx, msg, k, govKeeper)
		// disabled other msg handling
		//case types.MsgEditValidator:
		//	return handleMsgEditValidator(ctx, msg, k)
		//case types.MsgDelegate:
		//	return handleMsgDelegate(ctx, msg, k)
		//case types.MsgBeginRedelegate:
		//	return handleMsgBeginRedelegate(ctx, msg, k)
		//case types.MsgBeginUnbonding:
		//	return handleMsgBeginUnbonding(ctx, msg, k)
		//case MsgSideChain
		case types.MsgCreateSideChainValidator:
			return handleMsgCreateSideChainValidator(ctx, msg, k)
		case types.MsgEditSideChainValidator:
			return handleMsgEditSideChainValidator(ctx, msg, k)
		case types.MsgSideChainDelegate:
			return handleMsgSideChainDelegate(ctx, msg, k)
		case types.MsgSideChainRedelegate:
			return handleMsgSideChainRedelegate(ctx, msg, k)
		case types.MsgSideChainUndelegate:
			return handleMsgSideChainUndelegate(ctx, msg, k)
		default:
			return sdk.ErrTxDecode("invalid message parse in staking module").Result()
		}
	}
}

func NewStakeHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		// NOTE msg already has validate basic run
		switch msg := msg.(type) {
		case types.MsgCreateValidator:
			return handleMsgCreateValidator(ctx, msg, k)
		case types.MsgEditValidator:
			return handleMsgEditValidator(ctx, msg, k)
		case types.MsgDelegate:
			return handleMsgDelegate(ctx, msg, k)
		case types.MsgBeginRedelegate:
			return handleMsgBeginRedelegate(ctx, msg, k)
		case types.MsgBeginUnbonding:
			return handleMsgBeginUnbonding(ctx, msg, k)
		default:
			return sdk.ErrTxDecode("invalid message parse in staking module").Result()
		}
	}
}

//_____________________________________________________________________

// These functions assume everything has been authenticated,
// now we just perform action and save

func handleMsgCreateValidatorAfterProposal(ctx sdk.Context, msg MsgCreateValidatorProposal, k keeper.Keeper, govKeeper gov.Keeper) sdk.Result {
	height := ctx.BlockHeader().Height
	// do not checkProposal for the genesis txs
	if height != 0 {
		if err := checkCreateProposal(ctx, k, govKeeper, msg); err != nil {
			return ErrInvalidProposal(k.Codespace(), err.Error()).Result()
		}
	}

	return handleMsgCreateValidator(ctx, msg.MsgCreateValidator, k)
}

func handleMsgRemoveValidatorAfterProposal(ctx sdk.Context, msg MsgRemoveValidator, k keeper.Keeper, govKeeper gov.Keeper) sdk.Result {
	if err := checkRemoveProposal(ctx, k, govKeeper, msg); err != nil {
		return ErrInvalidProposal(k.Codespace(), err.Error()).Result()
	}

	var tags sdk.Tags
	var result sdk.Result
	k.IterateDelegationsToValidator(ctx, msg.ValAddr, func(del sdk.Delegation) (stop bool) {
		msgBeginUnbonding := MsgBeginUnbonding{
			ValidatorAddr: del.GetValidatorAddr(),
			DelegatorAddr: del.GetDelegatorAddr(),
			SharesAmount:  del.GetShares(),
		}
		result = handleMsgBeginUnbonding(ctx, msgBeginUnbonding, k)
		// handleMsgBeginUnbonding return error, abort execution
		if !result.IsOK() {
			return true
		}
		tags = tags.AppendTags(result.Tags)
		return false
	})

	// If there is a failure in handling MsgBeginUnbonding, return an error
	if !result.IsOK() {
		return result
	}

	return sdk.Result{Tags: tags}
}

func handleMsgCreateValidator(ctx sdk.Context, msg MsgCreateValidator, k keeper.Keeper) sdk.Result {
	// check to see if the pubkey or sender has been registered before
	_, found := k.GetValidator(ctx, msg.ValidatorAddr)
	if found {
		return ErrValidatorOwnerExists(k.Codespace()).Result()
	}

	_, found = k.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(msg.PubKey))
	if found {
		return ErrValidatorPubKeyExists(k.Codespace()).Result()
	}

	if msg.Delegation.Denom != k.BondDenom(ctx) {
		return ErrBadDenom(k.Codespace()).Result()
	}

	// self-delegate address will be used to collect fees.
	feeAddr := msg.DelegatorAddr
	validator := NewValidatorWithFeeAddr(feeAddr, msg.ValidatorAddr, msg.PubKey, msg.Description)
	commission := NewCommissionWithTime(
		msg.Commission.Rate, msg.Commission.MaxRate,
		msg.Commission.MaxChangeRate, ctx.BlockHeader().Time,
	)
	validator, err := validator.SetInitialCommission(commission)
	if err != nil {
		return err.Result()
	}

	k.SetValidator(ctx, validator)
	k.SetValidatorByConsAddr(ctx, validator)
	k.SetNewValidatorByPowerIndex(ctx, validator)

	k.OnValidatorCreated(ctx, validator.OperatorAddr)

	// move coins from the msg.Address account to a (self-delegation) delegator account
	// the validator account and global shares are updated within here
	_, err = k.Delegate(ctx, msg.DelegatorAddr, msg.Delegation, validator, true)
	if err != nil {
		return err.Result()
	}

	tags := sdk.NewTags(
		tags.DstValidator, []byte(msg.ValidatorAddr.String()),
		tags.Moniker, []byte(msg.Description.Moniker),
		tags.Identity, []byte(msg.Description.Identity),
	)

	return sdk.Result{
		Tags: tags,
	}
}

func checkCreateProposal(ctx sdk.Context, keeper keeper.Keeper, govKeeper gov.Keeper, msg MsgCreateValidatorProposal) error {
	proposal := govKeeper.GetProposal(ctx, msg.ProposalId)
	if proposal == nil {
		return fmt.Errorf("proposal %d does not exist", msg.ProposalId)
	}
	if proposal.GetProposalType() != gov.ProposalTypeCreateValidator {
		return fmt.Errorf("proposal type %s is not equal to %s",
			proposal.GetProposalType().String(), gov.ProposalTypeCreateValidator.String())
	}
	if proposal.GetStatus() != gov.StatusPassed {
		return fmt.Errorf("proposal status %s is not not passed",
			proposal.GetStatus().String())
	}

	var createValidatorJson CreateValidatorJsonMsg
	err := json.Unmarshal([]byte(proposal.GetDescription()), &createValidatorJson)
	if err != nil {
		return fmt.Errorf("unmarshal createValidator params failed, err=%s", err.Error())
	}
	createValidatorMsgProposal, err := createValidatorJson.ToMsgCreateValidator()
	if err != nil {
		return fmt.Errorf("invalid pubkey, err=%s", err.Error())
	}

	if !msg.MsgCreateValidator.Equals(createValidatorMsgProposal) {
		return fmt.Errorf("createValidator msg is not identical to the proposal one")
	}

	return nil
}

func checkRemoveProposal(ctx sdk.Context, keeper keeper.Keeper, govKeeper gov.Keeper, msg MsgRemoveValidator) error {
	proposal := govKeeper.GetProposal(ctx, msg.ProposalId)
	if proposal == nil {
		return fmt.Errorf("proposal %d does not exist", msg.ProposalId)
	}
	if proposal.GetProposalType() != gov.ProposalTypeRemoveValidator {
		return fmt.Errorf("proposal type %s is not equal to %s",
			proposal.GetProposalType().String(), gov.ProposalTypeRemoveValidator.String())
	}
	if proposal.GetStatus() != gov.StatusPassed {
		return fmt.Errorf("proposal status %s is not not passed",
			proposal.GetStatus().String())
	}

	// Check proposal description
	var proposalRemoveValidator MsgRemoveValidator
	err := json.Unmarshal([]byte(proposal.GetDescription()), &proposalRemoveValidator)
	if err != nil {
		return fmt.Errorf("unmarshal removeValidator params failed, err=%s", err.Error())
	}
	if !msg.ValAddr.Equals(proposalRemoveValidator.ValAddr) || !msg.ValConsAddr.Equals(proposalRemoveValidator.ValConsAddr) {
		return fmt.Errorf("removeValidator msg is not identical to the proposal one")
	}

	// Check validator information
	validatorToRemove, ok := keeper.GetValidator(ctx, msg.ValAddr)
	if !ok {
		return fmt.Errorf("trying to remove a non-existing validator")
	}
	if !validatorToRemove.ConsAddress().Equals(msg.ValConsAddr) {
		return fmt.Errorf("consensus address can't match actual validator consensus address")
	}

	// Check launcher authority
	if sdk.ValAddress(msg.LauncherAddr).Equals(msg.ValAddr) {
		return nil
	}
	// If the launcher isn't the target validator operator, then the launcher must be the operator of other active validator
	launcherValidator, ok := keeper.GetValidator(ctx, sdk.ValAddress(msg.LauncherAddr))
	if !ok {
		return fmt.Errorf("the launcher is not a validator operator")
	}
	if launcherValidator.Status != sdk.Bonded {
		return fmt.Errorf("the status of launcher validator is not bonded")
	}
	return nil
}

func handleMsgEditValidator(ctx sdk.Context, msg types.MsgEditValidator, k keeper.Keeper) sdk.Result {
	// validator must already be registered
	validator, found := k.GetValidator(ctx, msg.ValidatorAddr)
	if !found {
		return ErrNoValidatorFound(k.Codespace()).Result()
	}

	// replace all editable fields (clients should autofill existing values)
	description, err := validator.Description.UpdateDescription(msg.Description)
	if err != nil {
		return err.Result()
	}

	validator.Description = description

	if msg.CommissionRate != nil {
		commission, err := k.UpdateValidatorCommission(ctx, validator, *msg.CommissionRate)
		if err != nil {
			return err.Result()
		}
		validator.Commission = commission
		k.OnValidatorModified(ctx, msg.ValidatorAddr)
	}

	k.SetValidator(ctx, validator)

	tags := sdk.NewTags(
		tags.DstValidator, []byte(msg.ValidatorAddr.String()),
		tags.Moniker, []byte(description.Moniker),
		tags.Identity, []byte(description.Identity),
	)

	return sdk.Result{
		Tags: tags,
	}
}

func handleMsgDelegate(ctx sdk.Context, msg types.MsgDelegate, k keeper.Keeper) sdk.Result {
	validator, found := k.GetValidator(ctx, msg.ValidatorAddr)
	if !found {
		return ErrNoValidatorFound(k.Codespace()).Result()
	}

	if msg.Delegation.Denom != k.BondDenom(ctx) {
		return ErrBadDenom(k.Codespace()).Result()
	}

	if bytes.Equal(msg.DelegatorAddr.Bytes(), validator.OperatorAddr.Bytes()) {
		// if validator uses a different self-delegator address, the operator address is not allowed to delegate to itself.
		if !bytes.Equal(validator.OperatorAddr.Bytes(), validator.FeeAddr.Bytes()) {
			return ErrInvalidDelegator(k.Codespace()).Result()
		}
	}

	if validator.Jailed && !bytes.Equal(validator.FeeAddr, msg.DelegatorAddr) {
		return ErrValidatorJailed(k.Codespace()).Result()
	}

	_, err := k.Delegate(ctx, msg.DelegatorAddr, msg.Delegation, validator, true)
	if err != nil {
		return err.Result()
	}

	tags := sdk.NewTags(
		tags.Delegator, []byte(msg.DelegatorAddr.String()),
		tags.DstValidator, []byte(msg.ValidatorAddr.String()),
	)

	return sdk.Result{
		Tags: tags,
	}
}

func handleMsgBeginUnbonding(ctx sdk.Context, msg types.MsgBeginUnbonding, k keeper.Keeper) sdk.Result {
	ubd, err := k.BeginUnbonding(ctx, msg.DelegatorAddr, msg.ValidatorAddr, msg.SharesAmount)
	if err != nil {
		return err.Result()
	}

	finishTime := types.MsgCdc.MustMarshalBinaryLengthPrefixed(ubd.MinTime)

	tags := sdk.NewTags(
		tags.Delegator, []byte(msg.DelegatorAddr.String()),
		tags.SrcValidator, []byte(msg.ValidatorAddr.String()),
		tags.EndTime, finishTime,
	)
	return sdk.Result{Data: finishTime, Tags: tags}
}

func handleMsgBeginRedelegate(ctx sdk.Context, msg types.MsgBeginRedelegate, k keeper.Keeper) sdk.Result {
	red, err := k.BeginRedelegation(ctx, msg.DelegatorAddr, msg.ValidatorSrcAddr,
		msg.ValidatorDstAddr, msg.SharesAmount)
	if err != nil {
		return err.Result()
	}

	finishTime := types.MsgCdc.MustMarshalBinaryLengthPrefixed(red.MinTime)

	tags := sdk.NewTags(
		tags.Delegator, []byte(msg.DelegatorAddr.String()),
		tags.SrcValidator, []byte(msg.ValidatorSrcAddr.String()),
		tags.DstValidator, []byte(msg.ValidatorDstAddr.String()),
		tags.EndTime, finishTime,
	)
	return sdk.Result{Data: finishTime, Tags: tags}
}
