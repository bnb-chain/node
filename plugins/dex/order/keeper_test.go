package order

import (
	"os"
	"testing"
	"time"

	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/types"
	dextypes "github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
)

func MakeCodec() *wire.Codec {
	var cdc = wire.NewCodec()

	wire.RegisterCrypto(cdc) // Register crypto.
	bank.RegisterWire(cdc)
	sdk.RegisterWire(cdc) // Register Msgs
	tokens.RegisterWire(cdc)
	types.RegisterWire(cdc)
	cdc.RegisterConcrete(NewOrderMsg{}, "dex/NewOrder", nil)
	cdc.RegisterConcrete(CancelOrderMsg{}, "dex/CancelOrder", nil)

	cdc.RegisterConcrete(OrderBookSnapshot{}, "dex/OrderBookSnapshot", nil)
	cdc.RegisterConcrete(ActiveOrders{}, "dex/ActiveOrders", nil)

	return cdc
}
func TestKeeper_MarkBreatheBlock(t *testing.T) {
	cdc := MakeCodec()
	accountMapper := auth.NewAccountMapper(cdc, common.AccountStoreKey, types.ProtoAppAccount)
	coinKeeper := bank.NewKeeper(accountMapper)
	codespacer := sdk.NewCodespacer()
	keeper, _ := NewKeeper(common.DexStoreKey, coinKeeper, codespacer.RegisterNext(dextypes.DefaultCodespace), 2, cdc)
	assert := assert.New(t)
	memDB := db.NewMemDB()
	logger := log.NewTMLogger(os.Stdout)
	ms := store.NewCommitMultiStore(memDB).CacheMultiStore()
	ctx := sdk.NewContext(ms, abci.Header{}, true, logger)
	tt, _ := time.Parse(time.RFC3339, "2018-01-02T15:04:05Z")
	ts := tt.UnixNano() / 1000
	keeper.MarkBreatheBlock(42, ts, ctx)
	h := keeper.GetBreatheBlockHeight(tt, ctx.KVStore(common.DexStoreKey), 10)
	assert.Equal(42, h)
}
