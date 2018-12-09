package issue

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/log"
	"github.com/BiJie/BinanceChain/common/types"
	common "github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

// NewHandler creates a new token issue message handler
func NewHandler(tokenMapper store.Mapper, keeper bank.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		if msg, ok := msg.(IssueMsg); ok {
			return handleIssueToken(ctx, tokenMapper, keeper, msg)
		}

		errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
		return sdk.ErrUnknownRequest(errMsg).Result()
	}
}

func handleIssueToken(ctx sdk.Context, tokenMapper store.Mapper, keeper bank.Keeper, msg IssueMsg) sdk.Result {
	errLogMsg := "issue token failed"
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "token", "symbol", symbol, "name", msg.Name, "total_supply", msg.TotalSupply, "issuer", msg.From)
	var suffix string

	// TxHashKey is set in BaseApp's runMsgs
	txHash := ctx.Value(baseapp.TxHashKey)
	if txHashStr, ok := txHash.(string); ok {
		if len(txHashStr) >= types.TokenSymbolTxHashSuffixLen {
			suffix = txHashStr[:types.TokenSymbolTxHashSuffixLen]
		} else {
			logger.Error(errLogMsg,
				"reason", fmt.Sprintf("%s on Context had a length of %d, expected >= %d",
					baseapp.TxHashKey, len(txHashStr), types.TokenSymbolTxHashSuffixLen))
			return sdk.ErrInternal(fmt.Sprintf("unable to get the %s from Context", baseapp.TxHashKey)).Result()
		}
	} else {
		logger.Error(errLogMsg,
			"reason", fmt.Sprintf("%s on Context is not a string as expected", baseapp.TxHashKey))
		return sdk.ErrInternal(fmt.Sprintf("unable to get the %s from Context", baseapp.TxHashKey)).Result()
	}

	// the symbol is suffixed with the first n bytes of the tx hash
	symbol = fmt.Sprintf("%s-%s", symbol, suffix)

	if exists := tokenMapper.Exists(ctx, symbol); exists {
		logger.Info(errLogMsg, "reason", "already exists")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) already exists", msg.Symbol)).Result()
	}

	token, err := common.NewToken(msg.Name, symbol, msg.TotalSupply, msg.From)
	if err != nil {
		logger.Error(errLogMsg, "reason", "create token failed: "+err.Error())
	}

	if err := tokenMapper.NewToken(ctx, *token); err != nil {
		logger.Error(errLogMsg, "reason", "add token failed: "+err.Error())
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if _, _, sdkError := keeper.AddCoins(ctx, token.Owner,
		sdk.Coins{{
			Denom:  token.Symbol,
			Amount: token.TotalSupply.ToInt64(),
		}}); sdkError != nil {
		logger.Error(errLogMsg, "reason", "update balance failed: "+sdkError.Error())
		return sdkError.Result()
	}

	serialized, err := json.Marshal(token)
	if err != nil {
		logger.Error(errLogMsg, "reason", "unable to json serialize token: "+err.Error())
	}

	logger.Info("finished issuing token")

	return sdk.Result{
		Data: serialized,
		Log: fmt.Sprintf("Issued %s", token.Symbol),
	}
}
