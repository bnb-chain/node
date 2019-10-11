package list

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/dex/utils"
	"github.com/binance-chain/node/plugins/tokens"
)

// NewHandler initialises dex message handlers
func NewHandler(keeper *order.Keeper, tokenMapper tokens.Mapper, govKeeper gov.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case ListMsg:
			return handleList(ctx, keeper, tokenMapper, govKeeper, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized dex msg type: %v", reflect.TypeOf(msg).Name())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func checkListProposal(ctx sdk.Context, govKeeper gov.Keeper, msg ListMsg) error {
	proposal := govKeeper.GetProposal(ctx, msg.ProposalId)
	if proposal == nil {
		return fmt.Errorf("proposal %d does not exist", msg.ProposalId)
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

	if ctx.BlockHeader().Time.After(listParams.ExpireTime) {
		return fmt.Errorf("list time expired, expire_time=%s", listParams.ExpireTime.String())
	}

	if strings.ToUpper(msg.BaseAssetSymbol) != strings.ToUpper(listParams.BaseAssetSymbol) {
		return fmt.Errorf("base asset symbol(%s) is not identical to symbol in proposal(%s)",
			msg.BaseAssetSymbol, listParams.BaseAssetSymbol)
	}

	if strings.ToUpper(msg.QuoteAssetSymbol) != strings.ToUpper(listParams.QuoteAssetSymbol) {
		return fmt.Errorf("quote asset symbol(%s) is not identical to symbol in proposal(%s)",
			msg.QuoteAssetSymbol, listParams.QuoteAssetSymbol)
	}

	if msg.InitPrice != listParams.InitPrice {
		return fmt.Errorf("init price(%d) is not identical to price in proposal(%d)",
			msg.InitPrice, listParams.InitPrice)
	}

	return nil
}

func handleList(ctx sdk.Context, keeper *order.Keeper, tokenMapper tokens.Mapper, govKeeper gov.Keeper,
	msg ListMsg) sdk.Result {
	if err := checkListProposal(ctx, govKeeper, msg); err != nil {
		return types.ErrInvalidProposal(err.Error()).Result()
	}

	if err := keeper.CanListTradingPair(ctx, msg.BaseAssetSymbol, msg.QuoteAssetSymbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	baseToken, err := tokenMapper.GetToken(ctx, msg.BaseAssetSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if sdk.IsUpgrade(upgrade.ListingRuleUpgrade) {
		quoteToken, err := tokenMapper.GetToken(ctx, msg.QuoteAssetSymbol)
		if err != nil {
			return sdk.ErrInvalidCoins(err.Error()).Result()
		}

		if !baseToken.IsOwner(msg.From) && !quoteToken.IsOwner(msg.From) {
			return sdk.ErrUnauthorized("only the owner of the base asset or quote asset can list the trading pair").Result()
		}
	} else {
		if !tokenMapper.Exists(ctx, msg.QuoteAssetSymbol) {
			return sdk.ErrInvalidCoins("quote token does not exist").Result()
		}

		if !baseToken.IsOwner(msg.From) {
			return sdk.ErrUnauthorized("only the owner of the token can list the token").Result()
		}
	}

	if !tokenMapper.Exists(ctx, msg.QuoteAssetSymbol) {
		return sdk.ErrInvalidCoins("quote token does not exist").Result()
	}

	var lotSize int64
	if sdk.IsUpgrade(upgrade.LotSizeOptimization) {
		lotSize = keeper.DetermineLotSize(msg.BaseAssetSymbol, msg.QuoteAssetSymbol, msg.InitPrice)
	} else {
		lotSize = utils.CalcLotSize(msg.InitPrice)
	}
	pair := types.NewTradingPairWithLotSize(msg.BaseAssetSymbol, msg.QuoteAssetSymbol, msg.InitPrice, lotSize)
	err = keeper.PairMapper.AddTradingPair(ctx, pair)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	// this is done in memory! we must not run this block in checktx or simulate!
	if ctx.IsDeliverTx() { // only add engine during DeliverTx
		keeper.AddEngine(pair)
		log.With("module", "dex").Info("List new Pair and created new match engine", "pair", pair)
	}

	return sdk.Result{}
}
