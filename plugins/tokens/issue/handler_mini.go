package issue

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/binance-chain/node/common/log"
	common "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/tokens/store"
)

func handleIssueMiniToken(ctx sdk.Context, tokenMapper store.Mapper, bankKeeper bank.Keeper, msg IssueMiniMsg) sdk.Result {
	errLogMsg := "issue miniToken failed"
	origSymbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "mini-token", "symbol", origSymbol, "name", msg.Name, "total_supply", msg.TotalSupply, "issuer", msg.From)

	suffix, err := getTokenSuffix(ctx)
	if err != nil {
		logger.Error(errLogMsg, "reason", err.Error())
	}
	suffix += common.MiniTokenSymbolMSuffix
	symbol := fmt.Sprintf("%s-%s", origSymbol, suffix)

	if exists := tokenMapper.ExistsMini(ctx, symbol); exists {
		logger.Info(errLogMsg, "reason", "already exists")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) already exists", msg.Symbol)).Result()
	}

	token := common.NewMiniToken(msg.Name, origSymbol, symbol, common.MiniRangeType, msg.TotalSupply, msg.From, msg.Mintable, msg.TokenURI)
	return issue(ctx, logger, tokenMapper, bankKeeper, token)
}

func handleIssueTinyToken(ctx sdk.Context, tokenMapper store.Mapper, bankKeeper bank.Keeper, msg IssueTinyMsg) sdk.Result {
	errLogMsg := "issue tinyToken failed"
	origSymbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "mini-token", "symbol", origSymbol, "name", msg.Name, "total_supply", msg.TotalSupply, "issuer", msg.From)

	suffix, err := getTokenSuffix(ctx)
	if err != nil {
		logger.Error(errLogMsg, "reason", err.Error())
	}
	suffix += common.MiniTokenSymbolMSuffix
	symbol := fmt.Sprintf("%s-%s", origSymbol, suffix)

	if exists := tokenMapper.ExistsMini(ctx, symbol); exists {
		logger.Info(errLogMsg, "reason", "already exists")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) already exists", msg.Symbol)).Result()
	}

	token := common.NewMiniToken(msg.Name, origSymbol, symbol, common.TinyRangeType, msg.TotalSupply, msg.From, msg.Mintable, msg.TokenURI)
	return issue(ctx, logger, tokenMapper, bankKeeper, token)
}
