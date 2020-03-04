package cross_chain

import (
	"fmt"
	"strings"

	"github.com/binance-chain/node/common/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case BindMsg:
			return handleBindMsg(ctx, keeper, msg)
		case TransferMsg:
			return handleTransferMsg(ctx, keeper, msg)
		default:
			errMsg := "Unrecognized bridge msg type"
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleBindMsg(ctx sdk.Context, keeper Keeper, msg BindMsg) sdk.Result {
	symbol := strings.ToUpper(msg.Symbol)

	token, err := keeper.TokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", msg.Symbol)).Result()
	}

	if !token.IsOwner(msg.From) {
		return sdk.ErrUnauthorized(fmt.Sprintf("only the owner can bind token %s", msg.Symbol)).Result()
	}

	err = keeper.TokenMapper.UpdateBind(ctx, symbol, msg.ContractAddress.String(), msg.ContractDecimal)
	if err != nil {
		return sdk.ErrInternal(fmt.Sprintf("update token bind info error")).Result()
	}

	return sdk.Result{}
}

func handleTransferMsg(ctx sdk.Context, keeper Keeper, msg TransferMsg) sdk.Result {
	symbol := strings.ToUpper(msg.Amount.Denom)

	token, err := keeper.TokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", symbol)).Result()
	}

	if token.ContractAddress == "" {
		return ErrTokenNotBind(fmt.Sprintf("token %s is not bound", symbol)).Result()
	}

	_, cErr := keeper.BankKeeper.SendCoins(ctx, msg.From, types.PegAccount, sdk.Coins{msg.Amount})
	if cErr != nil {
		return cErr.Result()
	}

	return sdk.Result{}
}
