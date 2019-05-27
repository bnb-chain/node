package param

import (
	"fmt"

	"github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/wire"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
)

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
