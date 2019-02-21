package list

import (
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/tokens"
)

type ListHooks struct {
	orderKeeper *order.Keeper
	tokenMapper tokens.Mapper
}

func NewListHooks(orderKeeper *order.Keeper, tokenMapper tokens.Mapper) ListHooks {
	return ListHooks{
		orderKeeper: orderKeeper,
		tokenMapper: tokenMapper,
	}
}

var _ gov.GovHooks = ListHooks{}

func (hooks ListHooks) OnProposalSubmitted(ctx sdk.Context, proposal gov.Proposal) error {
	if proposal.GetProposalType() != gov.ProposalTypeListTradingPair {
		return nil
	}

	listParams := gov.ListTradingPairParams{}
	err := json.Unmarshal([]byte(proposal.GetDescription()), &listParams)
	if err != nil {
		return errors.New(fmt.Sprintf("unmarshal list params error, err=%s", err.Error()))
	}

	if listParams.BaseAssetSymbol == "" {
		return errors.New(fmt.Sprintf("base asset symbol should not be empty"))
	}

	if listParams.QuoteAssetSymbol == "" {
		return errors.New(fmt.Sprintf("quote asset symbol should not be empty"))
	}

	if listParams.InitPrice <= 0 {
		return errors.New("init price should larger than zero")
	}

	if listParams.ExpireTime.Before(ctx.BlockHeader().Time) {
		return errors.New("expire time should after now")
	}

	if hooks.orderKeeper.PairMapper.Exists(ctx, listParams.BaseAssetSymbol, listParams.QuoteAssetSymbol) ||
		hooks.orderKeeper.PairMapper.Exists(ctx, listParams.QuoteAssetSymbol, listParams.BaseAssetSymbol) {
		return errors.New(fmt.Sprintf("trading pair exists"))
	}

	if !hooks.tokenMapper.Exists(ctx, listParams.BaseAssetSymbol) {
		return errors.New(fmt.Sprintf("base token does not exist"))
	}

	if !hooks.tokenMapper.Exists(ctx, listParams.QuoteAssetSymbol) {
		return errors.New(fmt.Sprintf("quote token does not exist"))
	}

	return nil
}
