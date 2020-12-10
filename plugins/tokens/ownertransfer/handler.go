package ownertransfer

import (
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/plugins/tokens/store"
)

func NewHandler(tokenMapper store.Mapper, keeper bank.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case TransferOwnershipMsg:
			return handleOwnerTransfer(ctx, tokenMapper, msg)
		default:
			errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleOwnerTransfer(ctx sdk.Context, tokenMapper store.Mapper, msg TransferOwnershipMsg) sdk.Result {
	symbol := strings.ToUpper(msg.Symbol)
	logger := log.With("module", "token", "symbol", symbol, "from", msg.From, "newOwner", msg.NewOwner)

	token, err := tokenMapper.GetToken(ctx, symbol)
	if err != nil {
		logger.Info("transfer owner failed", "reason", "invalid token symbol")
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	if !token.IsOwner(msg.From) {
		logger.Info("transfer owner failed", "reason", "not token's owner", "from", msg.From, "owner", token.GetOwner())
		return sdk.ErrUnauthorized("only the owner of the token can transfer the owner").Result()
	}

	err = tokenMapper.UpdateOwner(ctx, symbol, msg.NewOwner)
	if err != nil {
		logger.Error("transfer owner failed", "reason", "update owner failed: "+err.Error())
		return sdk.ErrInternal(err.Error()).Result()
	}

	return sdk.Result{}
}
