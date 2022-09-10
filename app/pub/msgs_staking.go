package pub

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

// staking message
type StakingMsg struct {
	NumOfMsgs int
	Height    int64
	Timestamp int64

	Validators           []*Validator
	RemovedValidators    map[string][]sdk.ValAddress
	Delegations          map[string][]*Delegation
	UnbondingDelegations map[string][]*UnbondingDelegation
	ReDelegations        map[string][]*ReDelegation
	CompletedUBDs        map[string][]*CompletedUnbondingDelegation
	CompletedREDs        map[string][]*CompletedReDelegation
	DelegateEvents       map[string][]*DelegateEvent
	UndelegateEvents     map[string][]*UndelegateEvent
	RedelegateEvents     map[string][]*RedelegateEvent
	ElectedValidators    map[string][]*Validator
}

func (msg *StakingMsg) String() string {
	return fmt.Sprintf("StakingMsg at height: %d, numOfMsgs: %d", msg.Height, msg.NumOfMsgs)
}

func (msg *StakingMsg) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["numOfMsgs"] = msg.NumOfMsgs
	native["height"] = msg.Height
	native["timestamp"] = msg.Timestamp

	validators := make([]map[string]interface{}, len(msg.Validators))
	for id, v := range msg.Validators {
		validators[id] = v.toNativeMap()
	}
	//native["validators"] = validators
	native["validators"] = map[string]interface{}{"array": validators}

	removedValidators := make(map[string]interface{})
	for id, v := range msg.RemovedValidators {
		rvs := make([]string, len(v))
		for id, rv := range v {
			rvs[id] = rv.String()
		}
		removedValidators[id] = rvs
	}
	native["removedValidators"] = map[string]interface{}{"map": removedValidators}

	delegations := make(map[string]interface{})
	for chainId, v := range msg.Delegations {
		dels := make([]map[string]interface{}, len(v))
		for id, vv := range v {
			dels[id] = vv.toNativeMap()
		}
		delegations[chainId] = dels
	}
	//native["delegations"] = delegations
	native["delegations"] = map[string]interface{}{"map": delegations}

	unBondingDelegations := make(map[string]interface{})
	for chainId, v := range msg.UnbondingDelegations {
		ubds := make([]map[string]interface{}, len(v))
		for id, vv := range v {
			ubds[id] = vv.toNativeMap()
		}
		unBondingDelegations[chainId] = ubds
	}
	//native["unBondingDelegations"] = unBondingDelegations
	native["unBondingDelegations"] = map[string]interface{}{"map": unBondingDelegations}

	reDelegations := make(map[string]interface{})
	for chainId, v := range msg.ReDelegations {
		reds := make([]map[string]interface{}, len(v))
		for id, vv := range v {
			reds[id] = vv.toNativeMap()
		}
		reDelegations[chainId] = reds
	}
	//native["reDelegations"] = reDelegations
	native["reDelegations"] = map[string]interface{}{"map": reDelegations}

	completedUBDs := make(map[string]interface{})
	for chainId, v := range msg.CompletedUBDs {
		cubds := make([]map[string]interface{}, len(v))
		for id, vv := range v {
			cubds[id] = vv.toNativeMap()
		}
		completedUBDs[chainId] = cubds
	}
	//native["completedUBDs"] = completedUBDs
	native["completedUBDs"] = map[string]interface{}{"map": completedUBDs}

	completedREDs := make(map[string]interface{})
	for chainId, v := range msg.CompletedREDs {
		creds := make([]map[string]interface{}, len(v))
		for id, vv := range v {
			creds[id] = vv.toNativeMap()
		}
		completedREDs[chainId] = creds
	}
	//native["completedREDs"] = completedREDs
	native["completedREDs"] = map[string]interface{}{"map": completedREDs}

	delegateEvents := make(map[string]interface{})
	for chainId, v := range msg.DelegateEvents {
		deles := make([]map[string]interface{}, len(v))
		for id, vv := range v {
			deles[id] = vv.toNativeMap()
		}
		delegateEvents[chainId] = deles
	}
	native["delegateEvents"] = map[string]interface{}{"map": delegateEvents}

	unDelegateEvents := make(map[string]interface{})
	for chainId, v := range msg.UndelegateEvents {
		undeles := make([]map[string]interface{}, len(v))
		for id, vv := range v {
			undeles[id] = vv.toNativeMap()
		}
		unDelegateEvents[chainId] = undeles
	}
	native["unDelegateEvents"] = map[string]interface{}{"map": unDelegateEvents}

	reDelegateEvents := make(map[string]interface{})
	for chainId, v := range msg.RedelegateEvents {
		redeles := make([]map[string]interface{}, len(v))
		for id, vv := range v {
			redeles[id] = vv.toNativeMap()
		}
		reDelegateEvents[chainId] = redeles
	}
	native["reDelegateEvents"] = map[string]interface{}{"map": reDelegateEvents}

	electedValidators := make(map[string]interface{})
	for id, v := range msg.ElectedValidators {
		evs := make([]map[string]interface{}, len(v))
		for id, ev := range v {
			evs[id] = ev.toNativeMap()
		}
		electedValidators[id] = evs
	}
	native["electedValidators"] = map[string]interface{}{"map": electedValidators}

	return native
}

func (msg *StakingMsg) EssentialMsg() string {
	builder := strings.Builder{}
	fmt.Fprintf(&builder, "height:%d\n", msg.Height)
	if len(msg.Validators) > 0 {
		fmt.Fprintf(&builder, "validators: numOfMsg: %d\n", len(msg.Validators))
	}
	if len(msg.RemovedValidators) > 0 {
		fmt.Fprintf(&builder, "removed validators: numOfMsg: %d\n", len(msg.RemovedValidators))
	}
	if len(msg.Delegations) > 0 {
		fmt.Fprintf(&builder, "delegations:\n")
		for chainId, v := range msg.Delegations {
			fmt.Fprintf(&builder, "chainId:%s, numOfMsg: %d\n", chainId, len(v))
		}
	}
	if len(msg.UnbondingDelegations) > 0 {
		fmt.Fprintf(&builder, "unbondingDelegations:\n")
		for chainId, v := range msg.UnbondingDelegations {
			fmt.Fprintf(&builder, "chainId:%s, numOfMsg: %d\n", chainId, len(v))
		}
	}
	if len(msg.ReDelegations) > 0 {
		fmt.Fprintf(&builder, "reDelegations:\n")
		for chainId, v := range msg.ReDelegations {
			fmt.Fprintf(&builder, "chainId:%s, numOfMsg: %d\n", chainId, len(v))
		}
	}
	if len(msg.CompletedREDs) > 0 {
		fmt.Fprintf(&builder, "completedREDs:\n")
		for chainId, v := range msg.CompletedREDs {
			fmt.Fprintf(&builder, "chainId:%s, numOfMsg: %d\n", chainId, len(v))
		}
	}
	if len(msg.CompletedUBDs) > 0 {
		fmt.Fprintf(&builder, "completedUBDs:\n")
		for chainId, ubds := range msg.CompletedUBDs {
			fmt.Fprintf(&builder, "chainId:%s, numOfMsg: %d\n", chainId, len(ubds))
		}
	}
	if len(msg.DelegateEvents) > 0 {
		fmt.Fprintf(&builder, "delegateEvents:\n")
		for chainId, dels := range msg.DelegateEvents {
			fmt.Fprintf(&builder, "chainId:%s, numOfMsg: %d\n", chainId, len(dels))
		}
	}
	if len(msg.UndelegateEvents) > 0 {
		fmt.Fprintf(&builder, "undelegateEvents:\n")
		for chainId, udes := range msg.UndelegateEvents {
			fmt.Fprintf(&builder, "chainId:%s, numOfMsg: %d\n", chainId, len(udes))
		}
	}
	if len(msg.RedelegateEvents) > 0 {
		fmt.Fprintf(&builder, "redelegateEvents:\n")
		for chainId, rdes := range msg.RedelegateEvents {
			fmt.Fprintf(&builder, "chainId:%s, numOfMsg: %d\n", chainId, len(rdes))
		}
	}
	if len(msg.ElectedValidators) > 0 {
		fmt.Fprintf(&builder, "electedValidators:\n")
		for chainId, evs := range msg.ElectedValidators {
			fmt.Fprintf(&builder, "chainId:%s, numOfMsg: %d\n", chainId, len(evs))
		}
	}
	return builder.String()
}

func (msg *StakingMsg) EmptyCopy() AvroOrJsonMsg {
	return &StakingMsg{
		msg.NumOfMsgs,
		msg.Height,
		msg.Timestamp,
		make([]*Validator, 0),
		make(map[string][]sdk.ValAddress),
		make(map[string][]*Delegation),
		make(map[string][]*UnbondingDelegation),
		make(map[string][]*ReDelegation),
		make(map[string][]*CompletedUnbondingDelegation),
		make(map[string][]*CompletedReDelegation),
		make(map[string][]*DelegateEvent),
		make(map[string][]*UndelegateEvent),
		make(map[string][]*RedelegateEvent),
		make(map[string][]*Validator),
	}
}

type Validator stake.Validator

func (msg *Validator) String() string {
	return fmt.Sprintf("NewValidator: %v", msg.toNativeMap())
}

func (msg *Validator) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["feeAddr"] = msg.FeeAddr.String()
	native["operatorAddr"] = msg.OperatorAddr.String()
	if msg.ConsPubKey != nil {
		native["consAddr"] = sdk.ConsAddress(msg.ConsPubKey.Address()).String()
	}
	native["jailed"] = msg.Jailed

	native["status"] = sdk.BondStatusToString(msg.Status)
	native["tokens"] = msg.Tokens.RawInt()
	native["delegatorShares"] = msg.DelegatorShares.RawInt()

	description := make(map[string]interface{})
	description["moniker"] = msg.Description.Moniker
	description["identity"] = msg.Description.Identity
	description["website"] = msg.Description.Website
	description["details"] = msg.Description.Details
	native["description"] = description

	native["bondHeight"] = msg.BondHeight
	native["bondIntraTxCounter"] = int(msg.BondIntraTxCounter)

	commission := make(map[string]interface{})
	commission["rate"] = msg.Commission.Rate.RawInt()
	commission["maxRate"] = msg.Commission.MaxRate.RawInt()
	commission["maxChangeRate"] = msg.Commission.MaxChangeRate.RawInt()
	commission["updateTime"] = msg.Commission.UpdateTime.Unix()
	native["commission"] = commission

	native["distributionAddr"] = msg.DistributionAddr.String()
	native["sideChainId"] = msg.SideChainId
	native["sideConsAddr"] = sdk.HexAddress(msg.SideConsAddr)
	native["sideFeeAddr"] = sdk.HexAddress(msg.SideFeeAddr)

	return native
}

type Delegation stake.Delegation

func (msg *Delegation) String() string {
	return fmt.Sprintf("Delegate: %v", msg.toNativeMap())
}

func (msg *Delegation) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["delegator"] = msg.DelegatorAddr.String()
	native["validator"] = msg.ValidatorAddr.String()
	native["shares"] = msg.Shares.RawInt()
	return native
}

type UnbondingDelegation stake.UnbondingDelegation

func (msg *UnbondingDelegation) String() string {
	return fmt.Sprintf("UnDelegate: %v", msg.toNativeMap())
}

func (msg *UnbondingDelegation) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["delegator"] = msg.DelegatorAddr.String()
	native["validator"] = msg.ValidatorAddr.String()
	native["creationHeight"] = msg.CreationHeight
	initialBalance := Coin{
		Denom:  msg.InitialBalance.Denom,
		Amount: msg.InitialBalance.Amount,
	}
	native["initialBalance"] = initialBalance.ToNativeMap()
	balance := Coin{
		Denom:  msg.Balance.Denom,
		Amount: msg.Balance.Amount,
	}
	native["balance"] = balance.ToNativeMap()
	native["minTime"] = msg.MinTime.Unix()
	return native
}

type ReDelegation stake.Redelegation

func (msg *ReDelegation) String() string {
	return fmt.Sprintf("ReDelegate: %v", msg.toNativeMap())
}

func (msg *ReDelegation) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["delegator"] = msg.DelegatorAddr.String()
	native["srcValidator"] = msg.ValidatorSrcAddr.String()
	native["dstValidator"] = msg.ValidatorDstAddr.String()
	native["creationHeight"] = msg.CreationHeight
	native["sharesSrc"] = msg.SharesSrc.RawInt()
	native["sharesDst"] = msg.SharesDst.RawInt()
	initialBalance := Coin{
		Denom:  msg.InitialBalance.Denom,
		Amount: msg.InitialBalance.Amount,
	}
	native["initialBalance"] = initialBalance.ToNativeMap()
	balance := Coin{
		Denom:  msg.Balance.Denom,
		Amount: msg.Balance.Amount,
	}
	native["balance"] = balance.ToNativeMap()
	native["minTime"] = msg.MinTime.Unix()
	return native
}

type CompletedUnbondingDelegation struct {
	Validator sdk.ValAddress
	Delegator sdk.AccAddress
	Amount    Coin
}

func (msg *CompletedUnbondingDelegation) String() string {
	return fmt.Sprintf("CompletedUnbondingDelegation: %v", msg.toNativeMap())
}

func (msg *CompletedUnbondingDelegation) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["validator"] = msg.Validator.String()
	native["delegator"] = msg.Delegator.String()
	native["amount"] = msg.Amount.ToNativeMap()
	return native
}

type CompletedReDelegation struct {
	Delegator    sdk.AccAddress
	ValidatorSrc sdk.ValAddress
	ValidatorDst sdk.ValAddress
}

func (msg *CompletedReDelegation) String() string {
	return fmt.Sprintf("CompletedReDelegation: %v", msg.toNativeMap())
}

func (msg *CompletedReDelegation) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["delegator"] = msg.Delegator.String()
	native["srcValidator"] = msg.ValidatorSrc.String()
	native["dstValidator"] = msg.ValidatorDst.String()
	return native
}

type DelegateEvent struct {
	Delegator  sdk.AccAddress
	Validator  sdk.ValAddress
	Amount     Coin
	TxHash     string
	CrossStake bool
}

func (msg *DelegateEvent) String() string {
	return fmt.Sprintf("DelegateEvent: %v", msg.toNativeMap())
}

func (msg *DelegateEvent) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["delegator"] = msg.Delegator.String()
	native["validator"] = msg.Validator.String()
	native["amount"] = msg.Amount.ToNativeMap()
	native["txHash"] = msg.TxHash
	native["crossStake"] = msg.CrossStake
	return native
}

type UndelegateEvent struct {
	Delegator sdk.AccAddress
	Validator sdk.ValAddress
	Amount    Coin
	TxHash    string
}

func (msg *UndelegateEvent) String() string {
	return fmt.Sprintf("UndelegateEvent: %v", msg.toNativeMap())
}

func (msg *UndelegateEvent) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["delegator"] = msg.Delegator.String()
	native["validator"] = msg.Validator.String()
	native["amount"] = msg.Amount.ToNativeMap()
	native["txHash"] = msg.TxHash
	return native
}

type RedelegateEvent struct {
	Delegator    sdk.AccAddress
	ValidatorSrc sdk.ValAddress
	ValidatorDst sdk.ValAddress
	Amount       Coin
	TxHash       string
}

func (msg *RedelegateEvent) String() string {
	return fmt.Sprintf("RedelegateEvent: %v", msg.toNativeMap())
}

func (msg *RedelegateEvent) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["delegator"] = msg.Delegator.String()
	native["srcValidator"] = msg.ValidatorSrc.String()
	native["dstValidator"] = msg.ValidatorDst.String()
	native["amount"] = msg.Amount.ToNativeMap()
	native["txHash"] = msg.TxHash
	return native
}
