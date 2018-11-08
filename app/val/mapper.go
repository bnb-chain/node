package val

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/tendermint/crypto"
)

type Mapper interface {
	GetAccAddr(sdk.Context, crypto.Address) (sdk.AccAddress, error)
	SetVal(sdk.Context, crypto.Address, sdk.AccAddress)
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

func (m mapper) GetAccAddr(ctx sdk.Context, valAddr crypto.Address) (sdk.AccAddress, error) {
	store := ctx.KVStore(m.key)
	addr := store.Get(valAddr)
	if addr == nil {
		return nil, errors.New(fmt.Sprintf("valAddr(%X) not found", valAddr))
	}
	return sdk.AccAddress(addr), nil
}

func (m *mapper) SetVal(ctx sdk.Context, valAddr crypto.Address, addr sdk.AccAddress) {
	store := ctx.KVStore(m.key)
	store.Set(valAddr, addr)
}
