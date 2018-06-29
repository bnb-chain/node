package issue

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func NewHandler(tokenMapper store.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		if msg, ok := msg.(Msg); ok {
			return handleIssueToken(ctx, tokenMapper, accountMapper, keeper, msg)
		}

		errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
		return sdk.ErrUnknownRequest(errMsg).Result()
	}
}

func handleIssueToken(ctx sdk.Context, tokenMapper store.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper, msg Msg) sdk.Result {
	symbol := strings.ToUpper(msg.Symbol)
	exists := tokenMapper.Exists(ctx, symbol)
	if exists {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) already exists", msg.Symbol)).Result()
	}

	senderAccount := accountMapper.GetAccount(ctx, msg.From)
	// note here we need minus 1 because it was updated in anteHandler
	currentSequence := senderAccount.GetSequence() - 1

	token := types.NewToken(msg.Name, symbol, msg.Supply, msg.Decimal, msg.From)
	tokenAddr, err := types.GenerateTokenAddress(token, currentSequence)
	if err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}

	tokenAccount := accountMapper.GetAccount(ctx, tokenAddr)
	// this should not happen
	if tokenAccount != nil {
		return sdk.ErrInvalidAddress(fmt.Sprintf("duplicated token address(%X)", tokenAddr)).Result()
	}
	token.SetAddress(tokenAddr)

	err = tokenMapper.NewToken(ctx, token)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error()).Result()
	}

	// amount = supply * 10^decimals
	// TODO: need to fix Coin#Amount type to big.Int
	amount := int64(math.Pow10(int(token.Decimal))) * token.Supply

	_, _, sdkError := keeper.AddCoins(ctx, tokenAddr, append((sdk.Coins)(nil), sdk.Coin{Denom: token.Symbol, Amount: amount}))
	if sdkError != nil {
		return sdkError.Result()
	}

	return sdk.Result{}
}
