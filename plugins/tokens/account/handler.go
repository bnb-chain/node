package account

import (
	"reflect"

	common "github.com/binance-chain/node/common/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

// NewHandler creates a set account flags handler
func NewHandler(accKeeper auth.AccountKeeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case SetAccountFlagsMsg:
			return handleSetAccountFlags(ctx, accKeeper, msg)
		default:
			errMsg := "Unrecognized msg type: " + reflect.TypeOf(msg).Name()
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleSetAccountFlags(ctx sdk.Context, accKeeper auth.AccountKeeper, msg SetAccountFlagsMsg) sdk.Result {
	account, ok := accKeeper.GetAccount(ctx, msg.From).(common.NamedAccount)
	if !ok {
		return sdk.ErrInternal("unexpected account type").Result()
	}
	account.SetFlags(msg.Flags)
	accKeeper.SetAccount(ctx, account)
	return sdk.Result{}
}