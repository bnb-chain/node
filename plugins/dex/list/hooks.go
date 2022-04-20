package list

import (
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/tokens"
)

type ListHooks struct {
	orderKeeper *order.DexKeeper
	tokenMapper tokens.Mapper
}

func NewListHooks(orderKeeper *order.DexKeeper, tokenMapper tokens.Mapper) ListHooks {
	return ListHooks{
		orderKeeper: orderKeeper,
		tokenMapper: tokenMapper,
	}
}

var _ gov.GovHooks = ListHooks{}

func (hooks ListHooks) OnProposalSubmitted(ctx sdk.Context, proposal gov.Proposal) error {
	if proposal.GetProposalType() != gov.ProposalTypeListTradingPair {
		panic(fmt.Sprintf("received wrong type of proposal %x", proposal.GetProposalType()))
	}

	if sdk.IsUpgrade(upgrade.BEP142) {
		return errors.New("list trading pair proposal is disabled")
	}

	listParams := gov.ListTradingPairParams{}
	err := json.Unmarshal([]byte(proposal.GetDescription()), &listParams)
	if err != nil {
		return fmt.Errorf("unmarshal list params error, err=%s", err.Error())
	}

	if listParams.BaseAssetSymbol == "" {
		return errors.New("base asset symbol should not be empty")
	}

	if listParams.QuoteAssetSymbol == "" {
		return errors.New("quote asset symbol should not be empty")
	}

	if listParams.BaseAssetSymbol == listParams.QuoteAssetSymbol {
		return errors.New("base token and quote token should not be the same")
	}

	if listParams.InitPrice <= 0 {
		return errors.New("init price should larger than zero")
	}

	if listParams.ExpireTime.Before(ctx.BlockHeader().Time) {
		return errors.New("expire time should after now")
	}

	if !hooks.tokenMapper.ExistsBEP2(ctx, listParams.BaseAssetSymbol) {
		return errors.New("base token does not exist")
	}

	if !hooks.tokenMapper.ExistsBEP2(ctx, listParams.QuoteAssetSymbol) {
		return errors.New("quote token does not exist")
	}

	if err := hooks.orderKeeper.CanListTradingPair(ctx, listParams.BaseAssetSymbol, listParams.QuoteAssetSymbol); err != nil {
		return err
	}

	return nil
}

type DelistHooks struct {
	orderKeeper *order.DexKeeper
}

func NewDelistHooks(orderKeeper *order.DexKeeper) DelistHooks {
	return DelistHooks{
		orderKeeper: orderKeeper,
	}
}

var _ gov.GovHooks = DelistHooks{}

func (hooks DelistHooks) OnProposalSubmitted(ctx sdk.Context, proposal gov.Proposal) error {
	if proposal.GetProposalType() != gov.ProposalTypeDelistTradingPair {
		panic(fmt.Sprintf("received wrong type of proposal %x", proposal.GetProposalType()))
	}

	if !sdk.IsUpgrade(upgrade.BEP6) {
		return fmt.Errorf("proposal type %s is not supported", gov.ProposalTypeDelistTradingPair)
	}

	delistParams := gov.DelistTradingPairParams{}
	err := json.Unmarshal([]byte(proposal.GetDescription()), &delistParams)
	if err != nil {
		return fmt.Errorf("unmarshal list params error, err=%s", err.Error())
	}

	if delistParams.BaseAssetSymbol == "" {
		return errors.New("base asset symbol should not be empty")
	}

	if delistParams.QuoteAssetSymbol == "" {
		return errors.New("quote asset symbol should not be empty")
	}

	if delistParams.BaseAssetSymbol == delistParams.QuoteAssetSymbol {
		return errors.New("base asset symbol and quote asset symbol should not be the same")
	}

	if delistParams.Justification == "" {
		return errors.New("justification should not be empty")
	}

	if delistParams.IsExecuted {
		return errors.New("is_executed should be false")
	}

	if err := hooks.orderKeeper.CanDelistTradingPair(ctx, delistParams.BaseAssetSymbol, delistParams.QuoteAssetSymbol); err != nil {
		return err
	}

	return nil
}
