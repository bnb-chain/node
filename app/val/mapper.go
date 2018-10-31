package val

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"
)

type Mapper interface {
	GetValAddr(sdk.Context, sdk.AccAddress) crypto.Address
	SetVal(sdk.Context, sdk.AccAddress, crypto.Address)
}

var _ Mapper = (*mapper)(nil)

type mapper struct {
	key sdk.StoreKey
}

func NewMapper(key sdk.StoreKey) *mapper {
	return &mapper{
		key: key,
	}
}

func (m mapper) GetValAddr(ctx sdk.Context, addr sdk.AccAddress) crypto.Address {
	store := ctx.KVStore(m.key)
	return crypto.Address(store.Get(addr))
}

func (m *mapper) SetVal(ctx sdk.Context, addr sdk.AccAddress, valAddr crypto.Address) {
	store := ctx.KVStore(m.key)
	store.Set(addr, valAddr)
}
