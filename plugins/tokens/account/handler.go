package account

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	common "github.com/binance-chain/node/common/types"
)

// NewHandler creates a set account flags handler
func NewHandler(accKeeper auth.AccountKeeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case SetAccountFlagsMsg:
			return handleSetAccountFlags(ctx, accKeeper, msg)
		default:
			errMsg := fmt.Sprintf("unrecognized message type: %T", msg)
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleSetAccountFlags(ctx sdk.Context, accKeeper auth.AccountKeeper, msg SetAccountFlagsMsg) sdk.Result {
	account, ok := accKeeper.GetAccount(ctx, msg.From).(common.NamedAccount)
	if !ok {
		return sdk.ErrInternal("unexpected account type").Result()
	}
	if account.GetFlags() == msg.Flags {
		return sdk.ErrInvalidAccountFlags("try to set the same flags").Result()
	}
	account.SetFlags(msg.Flags)
	accKeeper.SetAccount(ctx, account)
	return sdk.Result{}
}
