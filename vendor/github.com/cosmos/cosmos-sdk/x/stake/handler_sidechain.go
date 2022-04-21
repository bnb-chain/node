package stake

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake/keeper"
	"github.com/cosmos/cosmos-sdk/x/stake/tags"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

func handleMsgCreateSideChainValidator(ctx sdk.Context, msg MsgCreateSideChainValidator, k keeper.Keeper) sdk.Result {
	if scCtx, err := k.ScKeeper.PrepareCtxForSideChain(ctx, msg.SideChainId); err != nil {
		return ErrInvalidSideChainId(k.Codespace()).Result()
	} else {
		ctx = scCtx
	}

	// check to see if the pubkey or sender has been registered before
	_, found := k.GetValidator(ctx, msg.ValidatorAddr)
	if found {
		return ErrValidatorOwnerExists(k.Codespace()).Result()
	}

	_, found = k.GetValidatorBySideConsAddr(ctx, msg.SideConsAddr)
	if found {
		return ErrValidatorSideConsAddrExist(k.Codespace()).Result()
	}

	minSelfDelegation := k.MinSelfDelegation(ctx)
	if msg.Delegation.Amount < minSelfDelegation {
		return ErrBadDelegationAmount(DefaultCodespace,
			fmt.Sprintf("self delegation must not be less than %d", minSelfDelegation)).Result()
	}
	if msg.Delegation.Denom != k.GetParams(ctx).BondDenom {
		return ErrBadDenom(k.Codespace()).Result()
	}

	// self-delegate address will be used to collect fees.
	feeAddr := msg.DelegatorAddr
	validator := NewSideChainValidator(feeAddr, msg.ValidatorAddr, msg.Description, msg.SideChainId, msg.SideConsAddr, msg.SideFeeAddr)
	commission := NewCommissionWithTime(
		msg.Commission.Rate, msg.Commission.MaxRate,
		msg.Commission.MaxChangeRate, ctx.BlockHeader().Time,
	)
	var err sdk.Error
	validator, err = validator.SetInitialCommission(commission)
	if err != nil {
		return err.Result()
	}

	k.SetValidator(ctx, validator)
	k.SetValidatorByConsAddr(ctx, validator) // here consAddr is the sideConsAddr
	k.SetNewValidatorByPowerIndex(ctx, validator)
	k.OnValidatorCreated(ctx, validator.OperatorAddr)

	// move coins from the msg.Address account to a (self-delegation) delegator account
	// the validator account and global shares are updated within here
	_, err = k.Delegate(ctx, msg.DelegatorAddr, msg.Delegation, validator, true)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{
		Tags: sdk.NewTags(
			tags.DstValidator, []byte(msg.ValidatorAddr.String()),
			tags.Moniker, []byte(msg.Description.Moniker),
			tags.Identity, []byte(msg.Description.Identity),
		),
	}
}

func handleMsgEditSideChainValidator(ctx sdk.Context, msg MsgEditSideChainValidator, k keeper.Keeper) sdk.Result {
	if scCtx, err := k.ScKeeper.PrepareCtxForSideChain(ctx, msg.SideChainId); err != nil {
		return ErrInvalidSideChainId(k.Codespace()).Result()
	} else {
		ctx = scCtx
	}

	// validator must already be registered
	validator, found := k.GetValidator(ctx, msg.ValidatorAddr)
	if !found {
		return ErrNoValidatorFound(k.Codespace()).Result()
	}

	// replace all editable fields (clients should autofill existing values)
	if description, err := validator.Description.UpdateDescription(msg.Description); err != nil {
		return err.Result()
	} else {
		validator.Description = description
	}

	if msg.CommissionRate != nil {
		commission, err := k.UpdateValidatorCommission(ctx, validator, *msg.CommissionRate)
		if err != nil {
			return err.Result()
		}
		validator.Commission = commission
		k.OnValidatorModified(ctx, msg.ValidatorAddr)
	}

	if len(msg.SideFeeAddr) != 0 {
		validator.SideFeeAddr = msg.SideFeeAddr
	}

	k.SetValidator(ctx, validator)
	return sdk.Result{
		Tags: sdk.NewTags(
			tags.DstValidator, []byte(msg.ValidatorAddr.String()),
			tags.Moniker, []byte(validator.Description.Moniker),
			tags.Identity, []byte(validator.Description.Identity),
		),
	}
}

func handleMsgSideChainDelegate(ctx sdk.Context, msg MsgSideChainDelegate, k keeper.Keeper) sdk.Result {
	if scCtx, err := k.ScKeeper.PrepareCtxForSideChain(ctx, msg.SideChainId); err != nil {
		return ErrInvalidSideChainId(k.Codespace()).Result()
	} else {
		ctx = scCtx
	}

	// we need this lower limit to prevent too many delegation records.
	minDelegationChange := k.MinDelegationChange(ctx)
	if msg.Delegation.Amount < minDelegationChange {
		return ErrBadDelegationAmount(DefaultCodespace, fmt.Sprintf("delegation must not be less than %d", minDelegationChange)).Result()
	}

	validator, found := k.GetValidator(ctx, msg.ValidatorAddr)
	if !found {
		return ErrNoValidatorFound(k.Codespace()).Result()
	}

	if msg.Delegation.Denom != k.BondDenom(ctx) {
		return ErrBadDenom(k.Codespace()).Result()
	}

	if err := checkOperatorAsDelegator(k, msg.DelegatorAddr, validator); err != nil {
		return err.Result()
	}

	// if the validator is jailed, only the self-delegator can delegate to itself
	if validator.Jailed && !bytes.Equal(validator.FeeAddr, msg.DelegatorAddr) {
		return ErrValidatorJailed(k.Codespace()).Result()
	}

	_, err := k.Delegate(ctx, msg.DelegatorAddr, msg.Delegation, validator, true)

	if err != nil {
		return err.Result()
	}

	// publish delegate event
	if k.PbsbServer != nil && ctx.IsDeliverTx() {
		event := types.SideDelegateEvent{
			DelegateEvent: types.DelegateEvent{
				StakeEvent: types.StakeEvent{
					IsFromTx: true,
				},
				Delegator: msg.DelegatorAddr,
				Validator: msg.ValidatorAddr,
				Amount:    msg.Delegation.Amount,
				Denom:     msg.Delegation.Denom,
				TxHash:    ctx.Value(baseapp.TxHashKey).(string),
			},
			SideChainId: msg.SideChainId,
		}
		k.PbsbServer.Publish(event)
	}

	return sdk.Result{
		Tags: sdk.NewTags(
			tags.Delegator, []byte(msg.DelegatorAddr.String()),
			tags.DstValidator, []byte(msg.ValidatorAddr.String()),
		),
	}
}

func handleMsgSideChainRedelegate(ctx sdk.Context, msg MsgSideChainRedelegate, k keeper.Keeper) sdk.Result {
	if scCtx, err := k.ScKeeper.PrepareCtxForSideChain(ctx, msg.SideChainId); err != nil {
		return ErrInvalidSideChainId(k.Codespace()).Result()
	} else {
		ctx = scCtx
	}

	if msg.Amount.Denom != k.BondDenom(ctx) {
		return ErrBadDenom(k.Codespace()).Result()
	}

	dstValidator, found := k.GetValidator(ctx, msg.ValidatorDstAddr)
	if !found {
		return types.ErrBadRedelegationDst(k.Codespace()).Result()
	}

	if err := checkOperatorAsDelegator(k, msg.DelegatorAddr, dstValidator); err != nil {
		return err.Result()
	}

	shares, err := k.ValidateUnbondAmount(ctx, msg.DelegatorAddr, msg.ValidatorSrcAddr, msg.Amount.Amount)
	if err != nil {
		return err.Result()
	}

	red, err := k.BeginRedelegation(ctx, msg.DelegatorAddr, msg.ValidatorSrcAddr,
		msg.ValidatorDstAddr, shares)
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

	// publish redelegate event
	if k.PbsbServer != nil && ctx.IsDeliverTx() {
		event := types.SideRedelegateEvent{
			RedelegateEvent: types.RedelegateEvent{
				StakeEvent: types.StakeEvent{
					IsFromTx: true,
				},
				Delegator:    msg.DelegatorAddr,
				SrcValidator: msg.ValidatorSrcAddr,
				DstValidator: msg.ValidatorDstAddr,
				Amount:       msg.Amount.Amount,
				Denom:        msg.Amount.Denom,
				TxHash:       ctx.Value(baseapp.TxHashKey).(string),
			},
			SideChainId: msg.SideChainId,
		}
		k.PbsbServer.Publish(event)
	}

	return sdk.Result{Data: finishTime, Tags: tags}
}

func handleMsgSideChainUndelegate(ctx sdk.Context, msg MsgSideChainUndelegate, k keeper.Keeper) sdk.Result {
	if scCtx, err := k.ScKeeper.PrepareCtxForSideChain(ctx, msg.SideChainId); err != nil {
		return ErrInvalidSideChainId(k.Codespace()).Result()
	} else {
		ctx = scCtx
	}

	if msg.Amount.Denom != k.BondDenom(ctx) {
		return ErrBadDenom(k.Codespace()).Result()
	}

	shares, err := k.ValidateUnbondAmount(ctx, msg.DelegatorAddr, msg.ValidatorAddr, msg.Amount.Amount)
	if err != nil {
		return err.Result()
	}

	ubd, err := k.BeginUnbonding(ctx, msg.DelegatorAddr, msg.ValidatorAddr, shares)
	if err != nil {
		return err.Result()
	}

	finishTime := types.MsgCdc.MustMarshalBinaryLengthPrefixed(ubd.MinTime)

	tags := sdk.NewTags(
		tags.Delegator, []byte(msg.DelegatorAddr.String()),
		tags.SrcValidator, []byte(msg.ValidatorAddr.String()),
		tags.EndTime, finishTime,
	)

	// publish undelegate event
	if k.PbsbServer != nil && ctx.IsDeliverTx() {
		event := types.SideUnDelegateEvent{
			UndelegateEvent: types.UndelegateEvent{
				StakeEvent: types.StakeEvent{
					IsFromTx: true,
				},
				Delegator: msg.DelegatorAddr,
				Validator: msg.ValidatorAddr,
				Amount:    msg.Amount.Amount,
				Denom:     msg.Amount.Denom,
				TxHash:    ctx.Value(baseapp.TxHashKey).(string),
			},
			SideChainId: msg.SideChainId,
		}
		k.PbsbServer.Publish(event)
	}

	return sdk.Result{Data: finishTime, Tags: tags}
}

// we allow the self-delegator delegating/redelegating to its validator.
// but the operator is not allowed if it is not a self-delegator
func checkOperatorAsDelegator(k Keeper, delegator sdk.AccAddress, validator Validator) sdk.Error {
	delegatorIsOperator := bytes.Equal(delegator.Bytes(), validator.OperatorAddr.Bytes())
	operatorIsSelfDelegator := validator.IsSelfDelegator(sdk.AccAddress(validator.OperatorAddr))

	if delegatorIsOperator && !operatorIsSelfDelegator {
		return ErrInvalidDelegator(k.Codespace())
	}
	return nil
}
