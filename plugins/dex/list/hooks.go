package list

import (
	"encoding/json"
	"errors"
	"fmt"

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

	if types.IsMiniTokenSymbol(listParams.BaseAssetSymbol) {
		if !hooks.tokenMapper.ExistsMini(ctx, listParams.BaseAssetSymbol) {
			return errors.New("base token does not exist")
		}
		if listParams.QuoteAssetSymbol != types.NativeTokenSymbol && listParams.QuoteAssetSymbol != order.BUSDSymbol {
			return errors.New("mini token can only be base symbol against BNB or BUSD")
		}
	} else {
		if types.IsMiniTokenSymbol(listParams.QuoteAssetSymbol) {
			return errors.New("mini token can not be listed as quote symbol")
		}
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

	lotSize := hooks.orderKeeper.DetermineLotSize(listParams.BaseAssetSymbol, listParams.QuoteAssetSymbol, listParams.InitPrice)

	pair := dextypes.NewTradingPairWithLotSize(listParams.BaseAssetSymbol, listParams.QuoteAssetSymbol, listParams.InitPrice, lotSize)
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
		if types.IsMiniTokenSymbol(listParams.BaseAssetSymbol) {
			if !hooks.tokenMapper.ExistsMini(ctx, listParams.BaseAssetSymbol) {
				return errors.New("base token does not exist")
			}
			if listParams.QuoteAssetSymbol != types.NativeTokenSymbol && listParams.QuoteAssetSymbol != order.BUSDSymbol {
				return errors.New("mini token can only be base symbol against BNB or BUSD")
			}
		} else {
			if types.IsMiniTokenSymbol(listParams.QuoteAssetSymbol) {
				return errors.New("mini token can not be listed as quote symbol")
			}
			if !hooks.tokenMapper.ExistsBEP2(ctx, listParams.BaseAssetSymbol) {
				return errors.New("base token does not exist")
			}

			if !hooks.tokenMapper.ExistsBEP2(ctx, listParams.QuoteAssetSymbol) {
				return errors.New("quote token does not exist")
			}
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
