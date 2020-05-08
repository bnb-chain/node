package issue

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/binance-chain/node/common/log"
	common "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/minitokens/store"
)

// NewHandler creates a new token issue message handler
func NewHandler(tokenMapper store.MiniTokenMapper, keeper bank.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case IssueMsg:
			return handleIssueToken(ctx, tokenMapper, keeper, msg)
		default:
			errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleIssueToken(ctx sdk.Context, tokenMapper store.MiniTokenMapper, bankKeeper bank.Keeper, msg IssueMsg) sdk.Result {
	errLogMsg := "issue miniToken failed"
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "miniToken", "symbol", symbol, "name", msg.Name, "total_supply", msg.TotalSupply, "issuer", msg.From)
	var suffix string

	if !sdk.IsUpgrade(upgrade.BEP8) {
		return sdk.ErrInternal(fmt.Sprint("issue miniToken is not supported at current height")).Result()
	}

	// TxHashKey is set in BaseApp's runMsgs
	txHash := ctx.Value(baseapp.TxHashKey)
	if txHashStr, ok := txHash.(string); ok {
		if len(txHashStr) >= common.MiniTokenSymbolTxHashSuffixLen {
			suffix = txHashStr[:common.MiniTokenSymbolTxHashSuffixLen] + common.MiniTokenSymbolMSuffix
		} else {
			logger.Error(errLogMsg,
				"reason", fmt.Sprintf("%s on Context had a length of %d, expected >= %d",
					baseapp.TxHashKey, len(txHashStr), common.MiniTokenSymbolTxHashSuffixLen))
			return sdk.ErrInternal(fmt.Sprintf("unable to get the %s from Context", baseapp.TxHashKey)).Result()
		}
	} else {
		logger.Error(errLogMsg,
			"reason", fmt.Sprintf("%s on Context is not a string as expected", baseapp.TxHashKey))
		return sdk.ErrInternal(fmt.Sprintf("unable to get the %s from Context", baseapp.TxHashKey)).Result()
	}

	if msg.TotalSupply < common.MiniTokenMinTotalSupply {
		logger.Info(errLogMsg, "reason", "total supply doesn't reach the min supply")
		return sdk.ErrInvalidCoins(fmt.Sprintf("total supply is too small, the min amount is %d",
			common.MiniTokenMinTotalSupply)).Result()
	}

	if msg.TotalSupply > common.MiniTokenSupplyUpperBound {
		logger.Info(errLogMsg, "reason", "total supply exceeds the max total supply")
		return sdk.ErrInvalidCoins(fmt.Sprintf("total supply is too large, the max total supply upperbound is %d",
			common.MiniTokenSupplyUpperBound)).Result()
	}
	// the symbol is suffixed with the first n bytes of the tx hash
	symbol = fmt.Sprintf("%s-%s", symbol, suffix)

	if !common.IsMiniTokenSymbol(symbol) {
		logger.Info(errLogMsg, "reason", "symbol not valid")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) is not valid for mini-token", symbol)).Result()
	}

	if exists := tokenMapper.Exists(ctx, symbol); exists {
		logger.Info(errLogMsg, "reason", "already exists")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) already exists", msg.Symbol)).Result()
	}

	token, err := common.NewMiniToken(msg.Name, symbol, msg.TokenType, msg.TotalSupply, msg.From, msg.Mintable, msg.TokenURI)
	if err != nil {
		logger.Error(errLogMsg, "reason", "create token failed: "+err.Error())
		return sdk.ErrInternal(fmt.Sprintf("unable to create token struct: %s", err.Error())).Result()
	}

	if err := tokenMapper.NewToken(ctx, *token); err != nil {
		logger.Error(errLogMsg, "reason", "add token failed: "+err.Error())
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if _, _, sdkError := bankKeeper.AddCoins(ctx, token.Owner,
		sdk.Coins{{
			Denom:  token.Symbol,
			Amount: token.TotalSupply.ToInt64(),
		}}); sdkError != nil {
		logger.Error(errLogMsg, "reason", "update balance failed: "+sdkError.Error())
		return sdkError.Result()
	}

	serialized, err := json.Marshal(token)
	if err != nil {
		logger.Error(errLogMsg, "reason", "fatal! unable to json serialize token: "+err.Error())
		panic(err) // fatal, the sky is falling in goland
	}

	logger.Info("finished issuing token")

	return sdk.Result{
		Data: serialized,
		Log:  fmt.Sprintf("Issued %s", token.Symbol),
	}
}
