package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

var _ auth.Account = (NamedAccount)(nil)

// TODO: maybe need to move GetFrozenCoins to the base interface
type NamedAccount interface {
	auth.Account
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
	auth.BaseAccount
	Name        string    `json:"name"`
	FrozenCoins sdk.Coins `json:"frozen"`
	LockedCoins sdk.Coins `json:"locked"`
}

// nolint
func (acc AppAccount) GetName() string                  { return acc.Name }
func (acc *AppAccount) SetName(name string)             { acc.Name = name }
func (acc AppAccount) GetFrozenCoins() sdk.Coins        { return acc.FrozenCoins }
func (acc *AppAccount) SetFrozenCoins(frozen sdk.Coins) { acc.FrozenCoins = frozen }
func (acc AppAccount) GetLockedCoins() sdk.Coins        { return acc.LockedCoins }
func (acc *AppAccount) SetLockedCoins(frozen sdk.Coins) { acc.LockedCoins = frozen }

// Get the AccountDecoder function for the custom AppAccount
func GetAccountDecoder(cdc *wire.Codec) auth.AccountDecoder {
	return func(accBytes []byte) (res auth.Account, err error) {
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

//___________________________________________________________________________________

// Genesis state - specify genesis trend
type DexGenesis struct {
	MakerFee             int64 `json:"makerFee"`
	TakerFee             int64 `json:"takerFee"`
	FeeFactor            int64 `json:"feeFactor"`
	MaxFee               int64 `json:"maxFee"`
	NativeTokenDiscount  int64 `json:"nativeTokenDiscount"`
	VolumeBucketDuration int64 `json:"volumeBucketDuration"`
}

// State to Unmarshal
type GenesisState struct {
	Accounts   []*GenesisAccount `json:"accounts"`
	DexGenesis DexGenesis        `json:"dex"`
}

// GenesisAccount doesn't need pubkey or sequence
type GenesisAccount struct {
	Name    string      `json:"name"`
	Address sdk.Address `json:"address"`
	Coins   sdk.Coins   `json:"coins"`
}

// NewGenesisAccount -
func NewGenesisAccount(aa *AppAccount) *GenesisAccount {
	return &GenesisAccount{
		Name:    aa.Name,
		Address: aa.GetAddress(),
		Coins:   aa.GetCoins().Sort(),
	}
}

// convert GenesisAccount to AppAccount
func (ga *GenesisAccount) ToAppAccount() (acc *AppAccount, err error) {
	baseAcc := auth.BaseAccount{
		Address: ga.Address,
		Coins:   ga.Coins.Sort(),
	}
	return &AppAccount{
		BaseAccount: baseAcc,
		Name:        ga.Name,
	}, nil
}
