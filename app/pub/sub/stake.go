package sub

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/pubsub"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

type CompletedUBD struct {
	Validator sdk.ValAddress
	Delegator sdk.AccAddress
	Amount    sdk.Coin
}

func SubscribeStakeEvent(sub *pubsub.Subscriber) error {
	err := sub.Subscribe(stake.Topic, func(event pubsub.Event) {
		switch e := event.(type) {
		case stake.SideDistributionEvent:
			sub.Logger.Debug(fmt.Sprintf("distribution event: %v \n", e))
			toPublish.EventData.StakeData.appendDistribution(e.SideChainId, e.Data)
		case stake.SideCompletedUBDEvent:
			sub.Logger.Debug(fmt.Sprintf("completed UBD event: %v \n", e))
			ubds := make([]CompletedUBD, len(e.CompUBDs))
			for i, ubd := range e.CompUBDs {
				cUBD := CompletedUBD{
					Validator: ubd.ValidatorAddr,
					Delegator: ubd.DelegatorAddr,
					Amount:    ubd.Balance,
				}
				ubds[i] = cUBD
			}
			toPublish.EventData.StakeData.appendCompletedUBD(e.SideChainId, ubds)
		case stake.SideCompletedREDEvent:
			sub.Logger.Debug(fmt.Sprintf("completed RED event: %v \n", e))
			toPublish.EventData.StakeData.appendCompletedRED(e.SideChainId, e.CompREDs)
		case stake.ValidatorUpdateEvent:
			sub.Logger.Debug(fmt.Sprintf("validator update event: %v \n", e))
			if len(e.Validator.SideChainId) == 0 { // ignore bbc validator update events
				return
			}
			if e.IsFromTx {
				stagingArea.StakeData.appendValidator(e.Validator)
			} else {
				toPublish.EventData.StakeData.appendValidator(e.Validator)
			}
		case stake.ValidatorRemovedEvent:
			sub.Logger.Debug(fmt.Sprintf("validator removed event: %v \n", e))
			chainId := e.SideChainId
			if len(chainId) == 0 {
				return // ignore bbc validator
			}
			if e.IsFromTx {
				stagingArea.StakeData.appendRemovedValidator(chainId, e.Operator)
			} else {
				toPublish.EventData.StakeData.appendRemovedValidator(chainId, e.Operator)
			}
		case stake.SideDelegationUpdateEvent:
			sub.Logger.Debug(fmt.Sprintf("delegation update event: %v \n", e))
			key := string(append(e.Delegation.DelegatorAddr.Bytes(), e.Delegation.ValidatorAddr.Bytes()...))
			if e.IsFromTx {
				stagingArea.StakeData.appendDelegation(e.SideChainId, key, e.Delegation)
			} else {
				toPublish.EventData.StakeData.appendDelegation(e.SideChainId, key, e.Delegation)
			}
		case stake.SideDelegationRemovedEvent:
			sub.Logger.Debug(fmt.Sprintf("delegation removed event: %v \n", e))
			if e.IsFromTx {
				stagingArea.StakeData.appendRemovedDelegation(e.SideChainId, e.DvPair)
			} else {
				toPublish.EventData.StakeData.appendRemovedDelegation(e.SideChainId, e.DvPair)
			}
		case stake.SideUBDUpdateEvent:
			sub.Logger.Debug(fmt.Sprintf("unbonding delegation update event: %v \n", e))
			key := string(append(e.UBD.DelegatorAddr.Bytes(), e.UBD.ValidatorAddr.Bytes()...))
			if e.IsFromTx {
				stagingArea.StakeData.appendUBD(e.SideChainId, key, e.UBD)
			} else {
				toPublish.EventData.StakeData.appendUBD(e.SideChainId, key, e.UBD)
			}
		case stake.SideREDUpdateEvent:
			sub.Logger.Debug(fmt.Sprintf("redelegation update event: %v \n", e))
			key := string(append(e.RED.DelegatorAddr.Bytes(), append(e.RED.ValidatorSrcAddr.Bytes(), e.RED.ValidatorDstAddr.Bytes()...)...))
			if e.IsFromTx {
				stagingArea.StakeData.appendRED(e.SideChainId, key, e.RED)
			} else {
				toPublish.EventData.StakeData.appendRED(e.SideChainId, key, e.RED)
			}
		case stake.SideDelegateEvent:
			sub.Logger.Debug(fmt.Sprintf("delegate event: %v \n", e))
			stagingArea.StakeData.appendDelegateEvent(e.SideChainId, e.DelegateEvent)
		case stake.SideUndelegateEvent:
			sub.Logger.Debug(fmt.Sprintf("undelegate event: %v \n", e))
			stagingArea.StakeData.appendUnDelegateEvent(e.SideChainId, e.UndelegateEvent)
		case stake.SideRedelegateEvent:
			sub.Logger.Debug(fmt.Sprintf("redelegate event: %v \n", e))
			stagingArea.StakeData.appendReDelegateEvent(e.SideChainId, e.RedelegateEvent)
		default:
			sub.Logger.Info("unknown event type")
		}
	})
	return err
}

type StakeData struct {
	// stash for stake topic
	Distribution         map[string][]stake.DistributionData             // ChainId -> []DistributionData
	CompletedUBDs        map[string][]CompletedUBD                       // ChainId -> []CompletedUBD
	CompletedREDs        map[string][]types.DVVTriplet                   // ChainId -> []DVVTriplet
	Validators           map[string]stake.Validator                      // operator(string) -> validator
	RemovedValidators    map[string][]sdk.ValAddress                     // ChainId -> []sdk.ValAddress
	Delegations          map[string]map[string]stake.Delegation          // ChainId -> delegator+validator -> Delegation
	RemovedDelegations   map[string][]types.DVPair                       // ChainId -> []DVPair
	UnbondingDelegations map[string]map[string]stake.UnbondingDelegation // ChainId -> delegator+validator -> UBD
	ReDelegations        map[string]map[string]stake.Redelegation        // ChainId -> delegator+srcValidator+dstValidator -> RED
	DelegateEvents       map[string][]stake.DelegateEvent                // ChainId -> delegate event
	UndelegateEvents     map[string][]stake.UndelegateEvent              // ChainId -> undelegate event
	RedelegateEvents     map[string][]stake.RedelegateEvent              // ChainId -> redelegate event
}

func (e *StakeData) appendDelegateEvent(chainId string, event stake.DelegateEvent) {
	if e.DelegateEvents == nil {
		e.DelegateEvents = make(map[string][]stake.DelegateEvent)
	}
	if _, ok := e.DelegateEvents[chainId]; !ok {
		e.DelegateEvents[chainId] = make([]stake.DelegateEvent, 0)
	}
	e.DelegateEvents[chainId] = append(e.DelegateEvents[chainId], event)
}

func (e *StakeData) appendDelegateEvents(chainId string, events []stake.DelegateEvent) {
	if e.DelegateEvents == nil {
		e.DelegateEvents = make(map[string][]stake.DelegateEvent)
	}
	if _, ok := e.DelegateEvents[chainId]; !ok {
		e.DelegateEvents[chainId] = make([]stake.DelegateEvent, 0)
	}
	e.DelegateEvents[chainId] = append(e.DelegateEvents[chainId], events...)
}

func (e *StakeData) appendUnDelegateEvent(chainId string, event stake.UndelegateEvent) {
	if e.UndelegateEvents == nil {
		e.UndelegateEvents = make(map[string][]stake.UndelegateEvent)
	}
	if _, ok := e.UndelegateEvents[chainId]; !ok {
		e.UndelegateEvents[chainId] = make([]stake.UndelegateEvent, 0)
	}
	e.UndelegateEvents[chainId] = append(e.UndelegateEvents[chainId], event)
}

func (e *StakeData) appendUnDelegateEvents(chainId string, events []stake.UndelegateEvent) {
	if e.UndelegateEvents == nil {
		e.UndelegateEvents = make(map[string][]stake.UndelegateEvent)
	}
	if _, ok := e.UndelegateEvents[chainId]; !ok {
		e.UndelegateEvents[chainId] = make([]stake.UndelegateEvent, 0)
	}
	e.UndelegateEvents[chainId] = append(e.UndelegateEvents[chainId], events...)
}

func (e *StakeData) appendReDelegateEvent(chainId string, event stake.RedelegateEvent) {
	if e.RedelegateEvents == nil {
		e.RedelegateEvents = make(map[string][]stake.RedelegateEvent)
	}
	if _, ok := e.RedelegateEvents[chainId]; !ok {
		e.RedelegateEvents[chainId] = make([]stake.RedelegateEvent, 0)
	}
	e.RedelegateEvents[chainId] = append(e.RedelegateEvents[chainId], event)
}

func (e *StakeData) appendReDelegateEvents(chainId string, events []stake.RedelegateEvent) {
	if e.RedelegateEvents == nil {
		e.RedelegateEvents = make(map[string][]stake.RedelegateEvent)
	}
	if _, ok := e.RedelegateEvents[chainId]; !ok {
		e.RedelegateEvents[chainId] = make([]stake.RedelegateEvent, 0)
	}
	e.RedelegateEvents[chainId] = append(e.RedelegateEvents[chainId], events...)
}

func (e *StakeData) appendDistribution(chainId string, data []stake.DistributionData) {
	if e.Distribution == nil {
		e.Distribution = make(map[string][]stake.DistributionData)
	}
	if _, ok := e.Distribution[chainId]; !ok {
		e.Distribution[chainId] = make([]stake.DistributionData, 0)
	}
	e.Distribution[chainId] = append(e.Distribution[chainId], data...)
}

func (e *StakeData) appendCompletedUBD(chainId string, ubds []CompletedUBD) {
	if e.CompletedUBDs == nil {
		e.CompletedUBDs = make(map[string][]CompletedUBD)
	}
	if _, ok := e.CompletedUBDs[chainId]; !ok {
		e.CompletedUBDs[chainId] = make([]CompletedUBD, 0)
	}
	e.CompletedUBDs[chainId] = append(e.CompletedUBDs[chainId], ubds...)
}

func (e *StakeData) appendCompletedRED(chainId string, reds []stake.DVVTriplet) {
	if e.CompletedREDs == nil {
		e.CompletedREDs = make(map[string][]stake.DVVTriplet)
	}
	if _, ok := e.CompletedREDs[chainId]; !ok {
		e.CompletedREDs[chainId] = make([]stake.DVVTriplet, 0)
	}
	e.CompletedREDs[chainId] = append(e.CompletedREDs[chainId], reds...)
}

func (e *StakeData) appendValidators(validators map[string]stake.Validator) {
	for _, v := range validators {
		e.appendValidator(v)
	}
}

func (e *StakeData) appendValidator(validator stake.Validator) {
	if e.Validators == nil {
		e.Validators = make(map[string]stake.Validator)
	}
	e.Validators[string(validator.OperatorAddr)] = validator
}

func (e *StakeData) appendRemovedValidators(chainId string, operators []sdk.ValAddress) {
	for _, v := range operators {
		e.appendRemovedValidator(chainId, v)
	}
}

func (e *StakeData) appendRemovedValidator(chainId string, operator sdk.ValAddress) {
	if e.RemovedValidators == nil {
		e.RemovedValidators = make(map[string][]sdk.ValAddress)
	}
	if e.RemovedValidators[chainId] == nil {
		e.RemovedValidators[chainId] = make([]sdk.ValAddress, 0)
	}
	e.RemovedValidators[chainId] = append(e.RemovedValidators[chainId], operator)
}

func (e *StakeData) appendDelegations(chainId string, delegations map[string]stake.Delegation) {
	for k, v := range delegations {
		e.appendDelegation(chainId, k, v)
	}
}

func (e *StakeData) appendDelegation(chainId string, key string, delegation stake.Delegation) {
	if e.Delegations == nil {
		e.Delegations = make(map[string]map[string]stake.Delegation)
	}
	if e.Delegations[chainId] == nil {
		e.Delegations[chainId] = make(map[string]stake.Delegation)
	}
	e.Delegations[chainId][key] = delegation
}

func (e *StakeData) appendRemovedDelegations(chainId string, pairs []stake.DVPair) {
	for _, pair := range pairs {
		e.appendRemovedDelegation(chainId, pair)
	}
}

func (e *StakeData) appendRemovedDelegation(chainId string, pair stake.DVPair) {
	if e.RemovedDelegations == nil {
		e.RemovedDelegations = make(map[string][]stake.DVPair)
	}
	if _, ok := e.RemovedDelegations[chainId]; !ok {
		e.RemovedDelegations[chainId] = make([]stake.DVPair, 0)
	}
	e.RemovedDelegations[chainId] = append(e.RemovedDelegations[chainId], pair)
}

func (e *StakeData) appendUBDs(chainId string, ubds map[string]stake.UnbondingDelegation) {
	for k, v := range ubds {
		e.appendUBD(chainId, k, v)
	}
}

func (e *StakeData) appendUBD(chainId string, key string, ubd stake.UnbondingDelegation) {
	if e.UnbondingDelegations == nil {
		e.UnbondingDelegations = make(map[string]map[string]stake.UnbondingDelegation)
	}
	if e.UnbondingDelegations[chainId] == nil {
		e.UnbondingDelegations[chainId] = make(map[string]stake.UnbondingDelegation)
	}
	e.UnbondingDelegations[chainId][key] = ubd
}

func (e *StakeData) appendREDs(chainId string, reds map[string]stake.Redelegation) {
	for k, v := range reds {
		e.appendRED(chainId, k, v)
	}
}

func (e *StakeData) appendRED(chainId string, key string, red stake.Redelegation) {
	if e.ReDelegations == nil {
		e.ReDelegations = make(map[string]map[string]stake.Redelegation)
	}
	if e.ReDelegations[chainId] == nil {
		e.ReDelegations[chainId] = make(map[string]stake.Redelegation)
	}
	e.ReDelegations[chainId][key] = red
}

func commitStake() {
	if len(stagingArea.StakeData.Distribution) > 0 {
		for chainId, v := range stagingArea.StakeData.Distribution {
			toPublish.EventData.StakeData.appendDistribution(chainId, v)
		}
	}
	if len(stagingArea.StakeData.DelegateEvents) > 0 {
		for chainId, v := range stagingArea.StakeData.DelegateEvents {
			toPublish.EventData.StakeData.appendDelegateEvents(chainId, v)
		}
	}
	if len(stagingArea.StakeData.UndelegateEvents) > 0 {
		for chainId, v := range stagingArea.StakeData.UndelegateEvents {
			toPublish.EventData.StakeData.appendUnDelegateEvents(chainId, v)
		}
	}
	if len(stagingArea.StakeData.RedelegateEvents) > 0 {
		for chainId, v := range stagingArea.StakeData.RedelegateEvents {
			toPublish.EventData.StakeData.appendReDelegateEvents(chainId, v)
		}
	}
	//if len(stagingArea.StakeData.CompletedUBDs) > 0 {
	//	for chainId, v := range stagingArea.StakeData.CompletedUBDs {
	//		toPublish.EventData.StakeData.appendCompletedUBD(chainId, v)
	//	}
	//}
	//if len(stagingArea.StakeData.CompletedREDs) > 0 {
	//	for chainId, v := range stagingArea.StakeData.CompletedREDs {
	//		toPublish.EventData.StakeData.appendCompletedRED(chainId, v)
	//	}
	//}
	if len(stagingArea.StakeData.Validators) > 0 {
		toPublish.EventData.StakeData.appendValidators(stagingArea.StakeData.Validators)
	}
	if len(stagingArea.StakeData.RemovedValidators) > 0 {
		for chainId, v := range stagingArea.StakeData.RemovedValidators {
			toPublish.EventData.StakeData.appendRemovedValidators(chainId, v)
		}
	}
	if len(stagingArea.StakeData.Delegations) > 0 {
		for chainId, v := range stagingArea.StakeData.Delegations {
			toPublish.EventData.StakeData.appendDelegations(chainId, v)
		}
	}
	if len(stagingArea.StakeData.RemovedDelegations) > 0 {
		for chainId, v := range stagingArea.StakeData.RemovedDelegations {
			toPublish.EventData.StakeData.appendRemovedDelegations(chainId, v)
		}
	}
	if len(stagingArea.StakeData.UnbondingDelegations) > 0 {
		for chainId, v := range stagingArea.StakeData.UnbondingDelegations {
			toPublish.EventData.StakeData.appendUBDs(chainId, v)
		}
	}
	if len(stagingArea.StakeData.ReDelegations) > 0 {
		for chainId, v := range stagingArea.StakeData.ReDelegations {
			toPublish.EventData.StakeData.appendREDs(chainId, v)
		}
	}
}
