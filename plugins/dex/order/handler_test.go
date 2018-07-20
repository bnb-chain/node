package order

// import (
// 	"bytes"
// 	"log"
// 	"testing"

// 	"github.com/BiJie/BinanceChain/common"
// 	bnt "github.com/BiJie/BinanceChain/common/types"
// 	"github.com/BiJie/BinanceChain/plugins/dex/store"
// 	"github.com/BiJie/BinanceChain/plugins/dex/types"
// 	tokenStore "github.com/BiJie/BinanceChain/plugins/tokens/store"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/cosmos/cosmos-sdk/x/auth"
// 	"github.com/cosmos/cosmos-sdk/x/bank"
// 	"github.com/stretchr/testify/assert"
// 	abci "github.com/tendermint/tendermint/abci/types"
// )

// func Test_handleNewOrder(t *testing.T) {
// 	assert := assert.New(t)
// 	var cdc = MakeCodec()
// 	// mappers
// 	accountMapper := auth.NewAccountMapper(cdc, common.AccountStoreKey, bnt.ProtoAppAccount)
// 	tokenMapper := tokenStore.NewMapper(cdc, common.TokenStoreKey)
// 	tradingPairMapper := store.NewTradingPairMapper(cdc, common.PairStoreKey)

// 	// Add handlers.
// 	coinKeeper := bank.NewKeeper(app.accountMapper)
// 	// TODO: make the concurrency configurable
// 	orderKeeper := NewOrderKeeper(common.DexStoreKey, app.coinKeeper, app.RegisterCodespace(types.DefaultCodespace), 2)

// 	var buf bytes.Buffer
// 	logger := log.New(&buf, "logger: ", log.Lshortfile)
// 	ctx := sdk.NewContext(memStore, abci.Header{ChainID: "1", Height: 1}, true, logger)

// }
