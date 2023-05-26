package issue

import (
	"encoding/json"
	"fmt"

	"reflect"
	"strconv"
	"strings"

	"github.com/bnb-chain/node/common/log"
	"github.com/bnb-chain/node/common/types"
	common "github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/common/upgrade"
	"github.com/bnb-chain/node/plugins/tokens/store"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

// NewHandler creates a new token issue message handler
func NewHandler(tokenMapper store.Mapper, keeper bank.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case IssueMsg:
			if sdk.IsUpgrade(upgrade.SecurityEnhancement) {
				return sdk.ErrMsgNotSupported("IssueMsg disabled in SecurityEnhancement upgrade").Result()
			}
			return handleIssueToken(ctx, tokenMapper, keeper, msg)
		case MintMsg:
			return handleMintToken(ctx, tokenMapper, keeper, msg)
		case IssueMiniMsg:
			if sdk.IsUpgrade(upgrade.SecurityEnhancement) {
				return sdk.ErrMsgNotSupported("IssueMiniMsg disabled in SecurityEnhancement upgrade").Result()
			}
			return handleIssueMiniToken(ctx, tokenMapper, keeper, msg)
		case IssueTinyMsg:
			if sdk.IsUpgrade(upgrade.SecurityEnhancement) {
				return sdk.ErrMsgNotSupported("IssueTinyMsg disabled in SecurityEnhancement upgrade").Result()
			}
			return handleIssueTinyToken(ctx, tokenMapper, keeper, msg)
		default:
			errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleIssueToken(ctx sdk.Context, tokenMapper store.Mapper, bankKeeper bank.Keeper, msg IssueMsg) sdk.Result {
	errLogMsg := "issue token failed"
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "token", "symbol", symbol, "name", msg.Name, "total_supply", msg.TotalSupply, "issuer", msg.From)
	suffix, err := getTokenSuffix(ctx)
	if err != nil {
		logger.Error(errLogMsg, "reason", err.Error())
		return sdk.ErrInternal(fmt.Sprintf("unable to get the %s from Context", baseapp.TxHashKey)).Result()
	}

	// the symbol is suffixed with the first n bytes of the tx hash
	symbol = fmt.Sprintf("%s-%s", symbol, suffix)
	if exists := tokenMapper.ExistsBEP2(ctx, symbol); exists {
		logger.Info(errLogMsg, "reason", "already exists")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) already exists", msg.Symbol)).Result()
	}

	token, err := common.NewToken(msg.Name, symbol, msg.TotalSupply, msg.From, msg.Mintable)
	if err != nil {
		logger.Error(errLogMsg, "reason", "create token failed: "+err.Error())
		return sdk.ErrInternal(fmt.Sprintf("unable to create token struct: %s", err.Error())).Result()
	}
	return issue(ctx, logger, tokenMapper, bankKeeper, token)
}

// Mint MiniToken is also handled by this function
func handleMintToken(ctx sdk.Context, tokenMapper store.Mapper, bankKeeper bank.Keeper, msg MintMsg) sdk.Result {
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "token", "symbol", symbol, "amount", msg.Amount, "minter", msg.From)

	errLogMsg := "mint token failed"
	token, err := tokenMapper.GetToken(ctx, symbol)
	if err != nil {
		logger.Info(errLogMsg, "reason", "symbol not exist")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", msg.Symbol)).Result()
	}

	if !token.IsMintable() {
		logger.Info(errLogMsg, "reason", "token cannot be minted")
		return sdk.ErrInvalidCoins(fmt.Sprintf("token(%s) cannot be minted", msg.Symbol)).Result()
	}

	if !token.IsOwner(msg.From) {
		logger.Info(errLogMsg, "reason", "not the token owner")
		return sdk.ErrUnauthorized(fmt.Sprintf("only the owner can mint token %s", msg.Symbol)).Result()
	}

	if common.IsMiniTokenSymbol(symbol) {
		miniToken := token.(*types.MiniToken)
		// use minus to prevent overflow
		if msg.Amount > miniToken.TokenType.UpperBound()-miniToken.TotalSupply.ToInt64() {
			logger.Info(errLogMsg, "reason", "total supply exceeds the max total supply")
			return sdk.ErrInvalidCoins(fmt.Sprintf("mint amount is too large, the max total supply is %d",
				miniToken.TokenType.UpperBound())).Result()
		}
	} else {
		// use minus to prevent overflow
		if msg.Amount > common.TokenMaxTotalSupply-token.GetTotalSupply().ToInt64() {
			logger.Info(errLogMsg, "reason", "exceed the max total supply")
			return sdk.ErrInvalidCoins(fmt.Sprintf("mint amount is too large, the max total supply is %ds",
				common.TokenMaxTotalSupply)).Result()
		}
	}

	newTotalSupply := token.GetTotalSupply().ToInt64() + msg.Amount
	err = tokenMapper.UpdateTotalSupply(ctx, symbol, newTotalSupply)
	if err != nil {
		logger.Error(errLogMsg, "reason", "update total supply failed: "+err.Error())
		return sdk.ErrInternal("update total supply failed").Result()
	}

	_, _, sdkError := bankKeeper.AddCoins(ctx, token.GetOwner(),
		sdk.Coins{{
			Denom:  token.GetSymbol(),
			Amount: msg.Amount,
		}})
	if sdkError != nil {
		logger.Error(errLogMsg, "reason", "update balance failed: "+sdkError.Error())
		return sdkError.Result()
	}

	logger.Info("finished minting token")
	return sdk.Result{
		Data: []byte(strconv.FormatInt(newTotalSupply, 10)),
	}
}

func issue(ctx sdk.Context, logger tmlog.Logger, tokenMapper store.Mapper, bankKeeper bank.Keeper, token common.IToken) sdk.Result {
	errLogMsg := "issue token failed"
	if err := tokenMapper.NewToken(ctx, token); err != nil {
		logger.Error(errLogMsg, "reason", "add token failed: "+err.Error())
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if _, _, sdkError := bankKeeper.AddCoins(ctx, token.GetOwner(),
		sdk.Coins{{
			Denom:  token.GetSymbol(),
			Amount: token.GetTotalSupply().ToInt64(),
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
		Log:  fmt.Sprintf("Issued %s", token.GetSymbol()),
	}
}

func getTokenSuffix(ctx sdk.Context) (suffix string, err error) {
	// TxHashKey is set in BaseApp's runMsgs
	txHash := ctx.Value(baseapp.TxHashKey)
	if txHashStr, ok := txHash.(string); ok {
		if len(txHashStr) >= types.TokenSymbolTxHashSuffixLen {
			suffix = txHashStr[:types.TokenSymbolTxHashSuffixLen]
			return suffix, nil
		} else {
			err = fmt.Errorf("%s on Context had a length of %d, expected >= %d",
				baseapp.TxHashKey, len(txHashStr), types.TokenSymbolTxHashSuffixLen)
			return "", err
		}
	} else {
		err = fmt.Errorf("%s on Context is not a string as expected", baseapp.TxHashKey)
		return "", err
	}
}
