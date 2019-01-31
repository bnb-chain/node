package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/binance-chain/node/wire"
)

var _ sdk.Account = (NamedAccount)(nil)

// TODO: maybe need to move GetFrozenCoins to the base interface
type NamedAccount interface {
	sdk.Account
	GetName() string
	SetName(string)

	GetFrozenCoins() sdk.Coins
	SetFrozenCoins(sdk.Coins)

	//TODO: this should merge into Coin
	GetLockedCoins() sdk.Coins
	SetLockedCoins(sdk.Coins)
}

// Custom extensions for this application.  This is just an example of
// extending auth.BaseAccount with custom fields.
//
// This is compatible with the stock auth.AccountStore, since
// auth.AccountStore uses the flexible go-amino library.

var _ NamedAccount = (*AppAccount)(nil)

type AppAccount struct {
	auth.BaseAccount `json:"base"`
	Name             string    `json:"name"`
	FrozenCoins      sdk.Coins `json:"frozen"`
	LockedCoins      sdk.Coins `json:"locked"`
}

// nolint
func (acc AppAccount) GetName() string                  { return acc.Name }
func (acc *AppAccount) SetName(name string)             { acc.Name = name }
func (acc AppAccount) GetFrozenCoins() sdk.Coins        { return acc.FrozenCoins }
func (acc *AppAccount) SetFrozenCoins(frozen sdk.Coins) { acc.FrozenCoins = frozen }
func (acc AppAccount) GetLockedCoins() sdk.Coins        { return acc.LockedCoins }
func (acc *AppAccount) SetLockedCoins(frozen sdk.Coins) { acc.LockedCoins = frozen }
func (acc *AppAccount) Clone() sdk.Account {
	baseAcc := acc.BaseAccount.Clone().(*auth.BaseAccount)
	clonedAcc := &AppAccount{
		BaseAccount: *baseAcc,
		Name:        acc.Name,
	}
	if acc.FrozenCoins == nil {
		clonedAcc.FrozenCoins = nil
	} else {
		coins := sdk.Coins{}
		for _, coin := range acc.FrozenCoins {
			coins = append(coins, sdk.Coin{Denom: coin.Denom, Amount: coin.Amount})
		}
		clonedAcc.FrozenCoins = coins
	}
	if acc.LockedCoins == nil {
		clonedAcc.LockedCoins = nil
	} else {
		coins := sdk.Coins{}
		for _, coin := range acc.LockedCoins {
			coins = append(coins, sdk.Coin{Denom: coin.Denom, Amount: coin.Amount})
		}
		clonedAcc.LockedCoins = coins
	}
	return clonedAcc
}

// Get the AccountDecoder function for the custom AppAccount
func GetAccountDecoder(cdc *wire.Codec) auth.AccountDecoder {
	return func(accBytes []byte) (res sdk.Account, err error) {
		if len(accBytes) == 0 {
			return nil, sdk.ErrTxDecode("accBytes are empty")
		}
		acct := new(AppAccount)
		err = cdc.UnmarshalBinaryBare(accBytes, &acct)
		if err != nil {
			panic(err)
		}
		return acct, err
	}
}

// Prototype function for AppAccount
func ProtoAppAccount() sdk.Account {
	aa := AppAccount{}
	return &aa
}
