package param

import (
	"fmt"

	"github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/wire"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
)

//---------------------    FeeChangeHooks -----------------
type FeeChangeHooks struct {
	cdc *wire.Codec
}

func NewFeeChangeHooks(cdc *wire.Codec) FeeChangeHooks {
	return FeeChangeHooks{cdc}
}

var _ gov.GovHooks = FeeChangeHooks{}

func (hooks FeeChangeHooks) OnProposalSubmitted(ctx sdk.Context, proposal gov.Proposal) error {
	if proposal.GetProposalType() != gov.ProposalTypeFeeChange {
		panic(fmt.Sprintf("received wrong type of proposal %x", proposal.GetProposalType()))
	}

	feeParams := types.FeeChangeParams{}
	err := hooks.cdc.UnmarshalJSON([]byte(proposal.GetDescription()), &feeParams)
	if err != nil {
		return fmt.Errorf("unmarshal feeParam error, err=%s", err.Error())
	}

	return feeParams.Check()
}

//---------------------    CSCParamsChangeHook  -----------------
type CSCParamsChangeHooks struct {
	cdc *wire.Codec
}

func NewCSCParamsChangeHook(cdc *wire.Codec) CSCParamsChangeHooks {
	return CSCParamsChangeHooks{cdc}
}

var _ gov.GovHooks = CSCParamsChangeHooks{}

func (hooks CSCParamsChangeHooks) OnProposalSubmitted(ctx sdk.Context, proposal gov.Proposal) error {
	if proposal.GetProposalType() != gov.ProposalTypeCSCParamsChange {
		panic(fmt.Sprintf("received wrong type of proposal %x", proposal.GetProposalType()))
	}

	var changeParam types.CSCParamChange
	strProposal := proposal.GetDescription()
	err := hooks.cdc.UnmarshalJSON([]byte(strProposal), &changeParam)
	if err != nil {
		return fmt.Errorf("get broken data when unmarshal CSCParamChange msg. proposalId %d, err %v", proposal.GetProposalID(), err)
	}
	return changeParam.Check()
}

//---------------------    SCParamsChangeHook  -----------------
type SCParamsChangeHooks struct {
	cdc *wire.Codec
}

func NewSCParamsChangeHook(cdc *wire.Codec) SCParamsChangeHooks {
	return SCParamsChangeHooks{cdc}
}

var _ gov.GovHooks = SCParamsChangeHooks{}

func (hooks SCParamsChangeHooks) OnProposalSubmitted(ctx sdk.Context, proposal gov.Proposal) error {
	if proposal.GetProposalType() != gov.ProposalTypeSCParamsChange {
		panic(fmt.Sprintf("received wrong type of proposal %x", proposal.GetProposalType()))
	}

	var changeParam types.SCChangeParams
	strProposal := proposal.GetDescription()
	err := hooks.cdc.UnmarshalJSON([]byte(strProposal), &changeParam)
	if err != nil {
		fmt.Errorf("get broken data when unmarshal SCParamsChange msg. proposalId %d, err %v", proposal.GetProposalID(), err)
	}
	return changeParam.Check()
}
