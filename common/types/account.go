package types

import (
	"github.com/BiJie/BinanceChain/plugins/dex"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

var _ sdk.Account = (NamedAccount)(nil)

// TODO: maybe need to move GetFrozenCoins to the base interface
type NamedAccount interface {
	sdk.Account
	GetName() string
	SetName(string)

	GetFrozenCoins() sdk.Coins
	SetFrozenCoins(sdk.Coins)
}

// Custom extensions for this application.  This is just an example of
// extending auth.BaseAccount with custom fields.
//
// This is compatible with the stock auth.AccountStore, since
// auth.AccountStore uses the flexible go-amino library.

var _ NamedAccount = (*AppAccount)(nil)

type AppAccount struct {
	auth.BaseAccount
	Name        string    `json:"name"`
	FrozenCoins sdk.Coins `json:"frozen"`
}

// nolint
func (acc AppAccount) GetName() string      { return acc.Name }
func (acc *AppAccount) SetName(name string) { acc.Name = name }

func (acc AppAccount) GetFrozenCoins() sdk.Coins        { return acc.FrozenCoins }
func (acc *AppAccount) SetFrozenCoins(frozen sdk.Coins) { acc.FrozenCoins = frozen }

// Get the AccountDecoder function for the custom NamedAccount
func GetAccountDecoder(cdc *wire.Codec) sdk.AccountDecoder {
	return func(accBytes []byte) (res sdk.Account, err error) {
		if len(accBytes) == 0 {
			return nil, sdk.ErrTxDecode("accBytes are empty")
		}
		acct := AppAccount{}
		err = cdc.UnmarshalBinaryBare(accBytes, &acct)
		if err != nil {
			panic(err)
		}
		return &acct, err
	}
}

//___________________________________________________________________________________

// State to Unmarshal
type GenesisState struct {
	Accounts   []*GenesisAccount `json:"accounts"`
	DexGenesis dex.DexGenesis    `json:"dex"`
}

// GenesisAccount doesn't need pubkey or sequence
type GenesisAccount struct {
	Name    string      `json:"name"`
	Address sdk.Address `json:"address"`
	Coins   sdk.Coins   `json:"coins"`
}

func NewGenesisAccount(aa NamedAccount) *GenesisAccount {
	return &GenesisAccount{
		Name:    aa.GetName(),
		Address: aa.GetAddress(),
		Coins:   aa.GetCoins().Sort(),
	}
}

// convert GenesisAccount to NamedAccount
func (ga *GenesisAccount) ToAppAccount() (acc NamedAccount, err error) {
	baseAcc := auth.BaseAccount{
		Address: ga.Address,
		Coins:   ga.Coins.Sort(),
	}
	return &AppAccount{
		BaseAccount: baseAcc,
		Name:        ga.Name,
	}, nil
}
