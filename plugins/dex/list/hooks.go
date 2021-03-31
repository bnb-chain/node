package list

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/dex/order"
	dextypes "github.com/binance-chain/node/plugins/dex/types"
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

var _ gov.ExtGovHooks = ListHooks{}

func (hooks ListHooks) OnProposalPassed(ctx sdk.Context, proposal gov.Proposal) error {
	if !sdk.IsUpgrade(upgrade.ListRefactor) {
		return nil
	}

	if proposal.GetProposalType() != gov.ProposalTypeListTradingPair {
		return fmt.Errorf("proposal type(%s) should be %s",
			proposal.GetProposalType(), gov.ProposalTypeListTradingPair)
	}

	if proposal.GetStatus() != gov.StatusPassed {
		return fmt.Errorf("proposal status(%s) should be Passed before you can list your token",
			proposal.GetStatus())
	}

	listParams := gov.ListTradingPairParams{}
	err := json.Unmarshal([]byte(proposal.GetDescription()), &listParams)
	if err != nil {
		return fmt.Errorf("illegal list params in proposal, params=%s", proposal.GetDescription())
	}

	baseAssetSymbol := strings.ToUpper(listParams.BaseAssetSymbol)
	quoteAssetSymbol := strings.ToUpper(listParams.QuoteAssetSymbol)

	if err := checkListingPairOnMainMarket(hooks, ctx, baseAssetSymbol, quoteAssetSymbol); err != nil {
		return err
	}

	lotSize := hooks.orderKeeper.DetermineLotSize(baseAssetSymbol, quoteAssetSymbol, listParams.InitPrice)

	pair := dextypes.NewTradingPairWithLotSize(baseAssetSymbol, quoteAssetSymbol, listParams.InitPrice, lotSize)
	err = hooks.orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	if err != nil {
		return err
	}

	hooks.orderKeeper.AddEngine(pair)
	log.With("module", "dex").Info("List new Pair and created new match engine", "pair", pair)

	return nil
}

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

	if sdk.IsUpgrade(upgrade.ListRefactor) {
		if err := checkListingPairOnMainMarket(hooks, ctx, strings.ToUpper(listParams.BaseAssetSymbol), strings.ToUpper(listParams.QuoteAssetSymbol)); err != nil {
			return err
		}
	} else {
		if !hooks.tokenMapper.ExistsBEP2(ctx, listParams.BaseAssetSymbol) {
			return errors.New("base token does not exist")
		}

		if !hooks.tokenMapper.ExistsBEP2(ctx, listParams.QuoteAssetSymbol) {
			return errors.New("quote token does not exist")
		}
	}

	if err := hooks.orderKeeper.CanListTradingPair(ctx, listParams.BaseAssetSymbol, listParams.QuoteAssetSymbol); err != nil {
		return err
	}

	return nil
}

/**
 * 1. Existence check
 * 2. Mini asset can only be Base Asset against BNB/BUSD
 * 3. Asset can only listed on one market:
 *    a. if quote is BNB, check if exists Base/BUSD listed in new market already
 *    b. if quote is BUSD, check if exists Base/BNB listed in new market already
 *    c. else, check if exists Quote/BNB listed in new market already
 */
func checkListingPairOnMainMarket(hooks ListHooks, ctx sdk.Context, BaseAssetSymbol string, QuoteAssetSymbol string) error {
	if types.IsMiniTokenSymbol(BaseAssetSymbol) {
		if !hooks.tokenMapper.ExistsMini(ctx, BaseAssetSymbol) {
			return errors.New("base token does not exist")
		}
		if QuoteAssetSymbol != types.NativeTokenSymbol && QuoteAssetSymbol != order.BUSDSymbol {
			return errors.New("mini token can only be base symbol against BNB or BUSD")
		}
	} else {
		if types.IsMiniTokenSymbol(QuoteAssetSymbol) {
			return errors.New("mini token can not be listed as quote symbol")
		}
		if !hooks.tokenMapper.ExistsBEP2(ctx, BaseAssetSymbol) {
			return errors.New("base token does not exist")
		}

		if !hooks.tokenMapper.ExistsBEP2(ctx, QuoteAssetSymbol) {
			return errors.New("quote token does not exist")
		}
	}
	if types.NativeTokenSymbol == QuoteAssetSymbol {
		if pair, err := hooks.orderKeeper.PairMapper.GetTradingPair(ctx, BaseAssetSymbol, order.BUSDSymbol); err == nil {
			// TODO check if pair type is new market, return err: one token can only be listed in one market
			log.Info(fmt.Sprintf("%s", pair)) // remove this log
		}
	} else if order.BUSDSymbol == QuoteAssetSymbol {
		if pair, err := hooks.orderKeeper.PairMapper.GetTradingPair(ctx, BaseAssetSymbol, types.NativeTokenSymbol); err == nil {
			// TODO check if pair type is new market, return err: one token can only be listed in one market
			log.Info(fmt.Sprintf("%s", pair)) // remove this log
		}
	} else {
		if pair, err := hooks.orderKeeper.PairMapper.GetTradingPair(ctx, QuoteAssetSymbol, types.NativeTokenSymbol); err == nil {
			// TODO check if pair type is new market, return err: one token can only be listed in one market
			log.Info(fmt.Sprintf("%s", pair)) // remove this log
		}
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

type PromotionHooks struct {
	orderKeeper *order.DexKeeper
}

func NewPromotionHooks(orderKeeper *order.DexKeeper) PromotionHooks {
	return PromotionHooks{
		orderKeeper: orderKeeper,
	}
}

func (hooks PromotionHooks) OnProposalSubmitted(ctx sdk.Context, proposal gov.Proposal) error {
	if proposal.GetProposalType() != gov.ProposalTypeListPromotion {
		return errors.New(fmt.Sprintf("received wrong type of proposal %x", proposal.GetProposalType()))
	}
	if !sdk.IsUpgrade(upgrade.TradingPairPromotion) {
		return fmt.Errorf("proposal type %s is not supported", gov.ProposalTypeListPromotion)
	}
	promotionParams := gov.ListPromotionParams{}
	err := json.Unmarshal([]byte(proposal.GetDescription()), &promotionParams)
	if err != nil {
		return fmt.Errorf("unmarshal list promotion params error, err=%s", err.Error())
	}

	if promotionParams.BaseAssetSymbol == "" {
		return errors.New("base asset symbol should not be empty")
	}

	if err := hooks.orderKeeper.CanPromoteTradingPair(ctx, promotionParams.BaseAssetSymbol); err != nil {
		return err
	}

	return nil
}
