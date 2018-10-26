package helpers

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

// EnsureSignBuild signs and build the transaction from the msg, taken from cosmos-sdk context helpers pkg (it's unexported there)
func EnsureSignBuild(ctx context.CoreContext, name string, msgs []sdk.Msg, cdc *wire.Codec) (tyBytes []byte, err error) {
	err = context.EnsureAccountExists(ctx, name)
	if err != nil {
		return nil, err
	}

	ctx, err = context.EnsureAccountNumber(ctx)
	if err != nil {
		return nil, err
	}
	// default to next sequence number if none provided
	ctx, err = context.EnsureSequence(ctx)
	if err != nil {
		return nil, err
	}

	var txBytes []byte

	keybase, err := keys.GetKeyBase()
	if err != nil {
		return nil, err
	}

	info, err := keybase.Get(name)
	if err != nil {
		return nil, err
	}
	var passphrase string
	// Only need a passphrase for locally-stored keys
	if info.GetType() == "local" {
		passphrase, err = ctx.GetPassphraseFromStdin(name)
		if err != nil {
			return nil, fmt.Errorf("Error fetching passphrase: %v", err)
		}
	}
	txBytes, err = ctx.SignAndBuild(name, passphrase, msgs, cdc)
	if err != nil {
		return nil, fmt.Errorf("Error signing transaction: %v", err)
	}

	return txBytes, err
}
