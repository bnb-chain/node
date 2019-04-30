package list

import (
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/plugins/dex/store"
	"github.com/binance-chain/node/plugins/tokens"
)

type ListHooks struct {
	pairMapper  store.TradingPairMapper
	tokenMapper tokens.Mapper
}

func NewListHooks(pairMapper store.TradingPairMapper, tokenMapper tokens.Mapper) ListHooks {
	return ListHooks{
		pairMapper:  pairMapper,
		tokenMapper: tokenMapper,
	}
}

var _ gov.GovHooks = ListHooks{}

func (hooks ListHooks) OnProposalSubmitted(ctx sdk.Context, proposal gov.Proposal) error {
	if proposal.GetProposalType() != gov.ProposalTypeListTradingPair {
		panic(fmt.Sprintf("received wrong type of proposal %x", proposal.GetProposalType()))
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

	if !hooks.tokenMapper.Exists(ctx, listParams.BaseAssetSymbol) {
		return errors.New("base token does not exist")
	}

	if !hooks.tokenMapper.Exists(ctx, listParams.QuoteAssetSymbol) {
		return errors.New("quote token does not exist")
	}

	if hooks.pairMapper.Exists(ctx, listParams.BaseAssetSymbol, listParams.QuoteAssetSymbol) ||
		hooks.pairMapper.Exists(ctx, listParams.QuoteAssetSymbol, listParams.BaseAssetSymbol) {
		return errors.New("trading pair exists")
	}

	if err := checkListPrerequisiteTradingPair(ctx, hooks.pairMapper, listParams.BaseAssetSymbol, listParams.QuoteAssetSymbol); err != nil {
		return err
	}

	return nil
}

type DelistHooks struct {
	pairMapper store.TradingPairMapper
}

func NewDelistHooks(pairMapper store.TradingPairMapper) DelistHooks {
	return DelistHooks{
		pairMapper: pairMapper,
	}
}

var _ gov.GovHooks = DelistHooks{}

func (hooks DelistHooks) OnProposalSubmitted(ctx sdk.Context, proposal gov.Proposal) error {
	if proposal.GetProposalType() != gov.ProposalTypeDelistTradingPair {
		panic(fmt.Sprintf("received wrong type of proposal %x", proposal.GetProposalType()))
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

	if delistParams.DelayedDays <= 0 {
		return errors.New("delayed days should be positive")
	}

	if delistParams.DelayedDays > gov.MaxDelayedDays {
		return fmt.Errorf("delayed days should not be larger than %d", gov.MaxDelayedDays)
	}

	if !hooks.pairMapper.Exists(ctx, delistParams.BaseAssetSymbol, delistParams.QuoteAssetSymbol) {
		return fmt.Errorf("trading pair %s_%s does not exist", delistParams.BaseAssetSymbol, delistParams.QuoteAssetSymbol)
	}

	if err := checkDelistPrerequisiteTradingPair(ctx, hooks.pairMapper, delistParams.BaseAssetSymbol, delistParams.QuoteAssetSymbol); err != nil {
		return err
	}

	return nil
}
