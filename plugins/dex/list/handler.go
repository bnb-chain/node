package list

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/common/log"
	commonTypes "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/store"
	"github.com/binance-chain/node/plugins/dex/types"
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

func checkProposal(ctx sdk.Context, govKeeper gov.Keeper, msg ListMsg) error {
	proposal := govKeeper.GetProposal(ctx, msg.ProposalId)
	if proposal == nil {
		return errors.New(fmt.Sprintf("proposal %d does not exist", msg.ProposalId))
	}

	if proposal.GetProposalType() != gov.ProposalTypeListTradingPair {
		return errors.New(fmt.Sprintf("proposal type(%s) shoule be %s",
			proposal.GetProposalType(), gov.ProposalTypeListTradingPair))
	}

	if proposal.GetStatus() != gov.StatusPassed {
		return errors.New(fmt.Sprintf("proposal status(%s) should be Passed before you can list your token",
			proposal.GetStatus()))
	}

	listParams := gov.ListTradingPairParams{}
	err := json.Unmarshal([]byte(proposal.GetDescription()), &listParams)
	if err != nil {
		return errors.New(fmt.Sprintf("illegal list params in proposal, params=%s", proposal.GetDescription()))
	}

	if ctx.BlockHeader().Time.After(listParams.ExpireTime) {
		return errors.New(fmt.Sprintf("list time expired, expire_time=%s", listParams.ExpireTime.String()))
	}

	if strings.ToUpper(msg.BaseAssetSymbol) != strings.ToUpper(listParams.BaseAssetSymbol) ||
		strings.ToUpper(msg.QuoteAssetSymbol) != strings.ToUpper(listParams.QuoteAssetSymbol) ||
		msg.InitPrice != listParams.InitPrice {
		return errors.New("list params are not identical to proposal")
	}

	return nil
}

func checkPrerequisiteTradingPair(ctx sdk.Context, pairMapper store.TradingPairMapper, baseAssetSymbol, quoteAssetSymbol string) error {
	// trading pair against native token should exist if quote token is not native token
	baseAssetSymbol = strings.ToUpper(baseAssetSymbol)
	quoteAssetSymbol = strings.ToUpper(quoteAssetSymbol)

	if baseAssetSymbol != commonTypes.NativeTokenSymbol &&
		quoteAssetSymbol != commonTypes.NativeTokenSymbol {

		if !pairMapper.Exists(ctx, baseAssetSymbol, commonTypes.NativeTokenSymbol) &&
			!pairMapper.Exists(ctx, commonTypes.NativeTokenSymbol, baseAssetSymbol) {
			return errors.New(
				fmt.Sprintf("Token %s should be listed against BNB before against %s",
					baseAssetSymbol, quoteAssetSymbol))
		}

		if !pairMapper.Exists(ctx, quoteAssetSymbol, commonTypes.NativeTokenSymbol) &&
			!pairMapper.Exists(ctx, commonTypes.NativeTokenSymbol, quoteAssetSymbol) {
			return errors.New(
				fmt.Sprintf("Token %s should be listed against BNB before listing %s against %s",
					quoteAssetSymbol, baseAssetSymbol, quoteAssetSymbol))
		}
	}
	return nil
}

func handleList(
	ctx sdk.Context, keeper *order.Keeper, tokenMapper tokens.Mapper, govKeeper gov.Keeper, msg ListMsg,
) sdk.Result {
	if err := checkProposal(ctx, govKeeper, msg); err != nil {
		return types.ErrInvalidProposal(err.Error()).Result()
	}

	if keeper.PairMapper.Exists(ctx, msg.BaseAssetSymbol, msg.QuoteAssetSymbol) ||
		keeper.PairMapper.Exists(ctx, msg.QuoteAssetSymbol, msg.BaseAssetSymbol) {
		return sdk.ErrInvalidCoins("trading pair exists").Result()
	}

	if err := checkPrerequisiteTradingPair(ctx, keeper.PairMapper, msg.BaseAssetSymbol, msg.QuoteAssetSymbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	baseToken, err := tokenMapper.GetToken(ctx, msg.BaseAssetSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if !baseToken.IsOwner(msg.From) {
		return sdk.ErrUnauthorized("only the owner of the token can list the token").Result()
	}

	if !tokenMapper.Exists(ctx, msg.QuoteAssetSymbol) {
		return sdk.ErrInvalidCoins("quote token does not exist").Result()
	}

	pair := types.NewTradingPair(msg.BaseAssetSymbol, msg.QuoteAssetSymbol, msg.InitPrice)
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
