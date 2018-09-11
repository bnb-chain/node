package account

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/tendermint/tendermint/crypto"

	"github.com/BiJie/BinanceChain/common/types"
)

var globalAccountNumberKey = []byte("globalAccountNumber")

// This Mapper encodes/decodes accounts using the
// go-amino (binary) encoding/decoding library.
type Mapper struct {

	// The (unexposed) key used to access the store from the Context.
	key sdk.StoreKey

	// The prototypical Account constructor.
	proto func() auth.Account

	// The wire codec for binary encoding/decoding of accounts.
	cdc *wire.Codec
}

// NewMapper returns a new sdk.Mapper that
// uses go-amino to (binary) encode and decode concrete sdk.Accounts.
// nolint
func NewMapper(cdc *wire.Codec, key sdk.StoreKey, proto func() auth.Account) Mapper {
	return Mapper{
		key:   key,
		proto: proto,
		cdc:   cdc,
	}
}

// Implaements sdk.Mapper.
func (am Mapper) NewAccountWithAddress(ctx types.Context, addr sdk.AccAddress) auth.Account {
	acc := am.proto()
	err := acc.SetAddress(addr)
	if err != nil {
		// Handle w/ #870
		panic(err)
	}
	err = acc.SetAccountNumber(am.GetNextAccountNumber(ctx))
	if err != nil {
		// Handle w/ #870
		panic(err)
	}
	return acc
}

// New Account
func (am Mapper) NewAccount(ctx types.Context, acc auth.Account) auth.Account {
	err := acc.SetAccountNumber(am.GetNextAccountNumber(ctx))
	if err != nil {
		// TODO: Handle with #870
		panic(err)
	}
	return acc
}

// Turn an address to key used to get it from the account store
func AddressStoreKey(addr sdk.AccAddress) []byte {
	return append([]byte("account:"), addr.Bytes()...)
}

// Implements sdk.Mapper.
func (am Mapper) GetAccount(ctx types.Context, addr sdk.AccAddress) auth.Account {
	store := ctx.KVStore(am.key)
	bz := store.Get(AddressStoreKey(addr))
	if bz == nil {
		return nil
	}
	acc := am.decodeAccount(bz)
	return acc
}

// Implements sdk.Mapper.
func (am Mapper) SetAccount(ctx types.Context, acc auth.Account) {
	addr := acc.GetAddress()
	store := ctx.KVStore(am.key)
	bz := am.encodeAccount(acc)
	store.Set(AddressStoreKey(addr), bz)
}

// Implements sdk.Mapper.
func (am Mapper) IterateAccounts(ctx types.Context, process func(auth.Account) (stop bool)) {
	store := ctx.KVStore(am.key)
	iter := sdk.KVStorePrefixIterator(store, []byte("account:"))
	for {
		if !iter.Valid() {
			return
		}
		val := iter.Value()
		acc := am.decodeAccount(val)
		if process(acc) {
			return
		}
		iter.Next()
	}
}

// Returns the PubKey of the account at address
func (am Mapper) GetPubKey(ctx types.Context, addr sdk.AccAddress) (crypto.PubKey, sdk.Error) {
	acc := am.GetAccount(ctx, addr)
	if acc == nil {
		return nil, sdk.ErrUnknownAddress(addr.String())
	}
	return acc.GetPubKey(), nil
}

// Returns the Sequence of the account at address
func (am Mapper) GetSequence(ctx types.Context, addr sdk.AccAddress) (int64, sdk.Error) {
	acc := am.GetAccount(ctx, addr)
	if acc == nil {
		return 0, sdk.ErrUnknownAddress(addr.String())
	}
	return acc.GetSequence(), nil
}

func (am Mapper) setSequence(ctx types.Context, addr sdk.AccAddress, newSequence int64) sdk.Error {
	acc := am.GetAccount(ctx, addr)
	if acc == nil {
		return sdk.ErrUnknownAddress(addr.String())
	}
	err := acc.SetSequence(newSequence)
	if err != nil {
		// Handle w/ #870
		panic(err)
	}
	am.SetAccount(ctx, acc)
	return nil
}

// Returns and increments the global account number counter
func (am Mapper) GetNextAccountNumber(ctx types.Context) int64 {
	var accNumber int64
	store := ctx.KVStore(am.key)
	bz := store.Get(globalAccountNumberKey)
	if bz == nil {
		accNumber = 0
	} else {
		err := am.cdc.UnmarshalBinary(bz, &accNumber)
		if err != nil {
			panic(err)
		}
	}

	bz = am.cdc.MustMarshalBinary(accNumber + 1)
	store.Set(globalAccountNumberKey, bz)

	return accNumber
}

//----------------------------------------
// misc.

func (am Mapper) encodeAccount(acc auth.Account) []byte {
	bz, err := am.cdc.MarshalBinaryBare(acc)
	if err != nil {
		panic(err)
	}
	return bz
}

func (am Mapper) decodeAccount(bz []byte) (acc auth.Account) {
	err := am.cdc.UnmarshalBinaryBare(bz, &acc)
	if err != nil {
		panic(err)
	}
	return
}
