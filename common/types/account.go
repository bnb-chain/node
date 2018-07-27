package types

import (
	"encoding/json"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/tendermint/tendermint/libs/bech32"
)

const Bech32PrefixAccAddr = "bnb"

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

func (acc AppAccount) String() string {
	res := struct {
		Name          string   `json:"name"`
		Address       string   `json:"address"`
		AccountNumber int64    `json:"id"`
		Sequence      int64    `json:"sequence"`
		Balances      balances `json:"balances"`
	}{
		Name:          acc.GetName(),
		Address:       getBnbAddr(acc.GetAddress()),
		AccountNumber: acc.GetAccountNumber(),
		Sequence:      acc.GetSequence(),
	}

	tokenMap := make(map[string]balance)
	for _, coin := range acc.GetCoins() {
		tokenMap[coin.Denom] = balance{Symbol: coin.Denom, Free: coin.Amount.Int64()}
	}

	for _, coin := range acc.GetFrozenCoins() {
		if t, ok := tokenMap[coin.Denom]; ok {
			t.Frozen = coin.Amount.Int64()
		} else {
			tokenMap[coin.Denom] = balance{Symbol: coin.Denom, Frozen: coin.Amount.Int64()}
		}
	}

	for _, coin := range acc.GetLockedCoins() {
		if t, ok := tokenMap[coin.Denom]; ok {
			t.Locked = coin.Amount.Int64()
		} else {
			tokenMap[coin.Denom] = balance{Symbol: coin.Denom, Locked: coin.Amount.Int64()}
		}
	}

	for _, value := range tokenMap {
		res.Balances = append(res.Balances, value)
	}

	sort.Sort(res.Balances)
	str, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		return "Invalid account"
	}

	return string(str)
}

type balance struct {
	Symbol string `json:"symbol"`
	Free   int64  `json:"free"`
	Frozen int64  `json:"frozen"`
	Locked int64  `json:"locked"`
}

type balances []balance

func (bs balances) Len() int           { return len(bs) }
func (bs balances) Less(i, j int) bool { return bs[i].Symbol < bs[j].Symbol }
func (bs balances) Swap(i, j int)      { bs[i], bs[j] = bs[j], bs[i] }

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

// Prototype function for AppAccount
func ProtoAppAccount() auth.Account {
	aa := AppAccount{}
	return &aa
}

func getBnbAddr(bz sdk.AccAddress) string {
	bech32Addr, err := bech32.ConvertAndEncode(Bech32PrefixAccAddr, bz.Bytes())
	if err != nil {
		panic(err)
	}

	return bech32Addr
}
