package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/plugins/dex/list"
)

type GovHooks struct {
	listHooks list.ListHooks
}

var _ gov.GovHooks = GovHooks{}

func NewGovHooks(listHooks list.ListHooks) GovHooks {
	return GovHooks{
		listHooks: listHooks,
	}
}

func (hooks GovHooks) OnProposalSubmitted(ctx sdk.Context, proposal gov.Proposal) error {
	err := hooks.listHooks.OnProposalSubmitted(ctx, proposal)
	if err != nil {
		return err
	}
	return nil
}
