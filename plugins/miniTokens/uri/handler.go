package uri

import (
	"fmt"
	"github.com/binance-chain/node/common/log"
	common "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/minitokens/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"reflect"
	"strings"
)

func NewHandler(tokenMapper store.MiniTokenMapper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case SetURIMsg:
			return handleSetURI(ctx, tokenMapper, msg)
		default:
			errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleSetURI(ctx sdk.Context, miniTokenMapper store.MiniTokenMapper, msg SetURIMsg) sdk.Result {
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "miniToken", "symbol", symbol, "tokenURI", msg.TokenURI, "from", msg.From)

	errLogMsg := "set token URI failed"
	token, err := miniTokenMapper.GetToken(ctx, symbol)
	if err != nil {
		logger.Info(errLogMsg, "reason", "symbol not exist")
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", msg.Symbol)).Result()
	}

	if !token.IsOwner(msg.From) {
		logger.Info(errLogMsg, "reason", "not the token owner")
		return sdk.ErrUnauthorized(fmt.Sprintf("only the owner can mint token %s", msg.Symbol)).Result()
	}

	if len(msg.TokenURI) < 1 {
		return sdk.ErrInvalidCoins(fmt.Sprintf("token uri should not exceed %v characters", common.MaxTokenURILength)).Result()
	}

	if len(msg.TokenURI) > common.MaxTokenURILength {
		return sdk.ErrInvalidCoins(fmt.Sprintf("token uri should not exceed %v characters", common.MaxTokenURILength)).Result()
	}
	err = miniTokenMapper.UpdateTokenURI(ctx, symbol, msg.TokenURI)
	if err != nil {
		logger.Error(errLogMsg, "reason", "update token uri failed: "+err.Error())
		return sdk.ErrInternal(fmt.Sprintf("update token uri failed")).Result()
	}


	logger.Info("finished update token uri")
	return sdk.Result{
		Data: []byte(msg.TokenURI),
	}
}
