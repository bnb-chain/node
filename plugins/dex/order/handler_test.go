package order

// import (
// 	"testing"

// 	o "github.com/BiJie/BinanceChain/plugins/dex/order"
// 	test "github.com/BiJie/BinanceChain/test/abci"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/stretchr/testify/assert"
// 	abci "github.com/tendermint/tendermint/abci/types"
// )

// func Test_handleNewOrder(t *testing.T) {
// 	assert := assert.New(t)
// 	var cdc = test.TA().GetCodec()
// 	// mappers
// 	accountMapper := test.TA().GetAccountMapper()
// 	tokenMapper := test.TA().GetTokenMapper()

// 	// Add handlers.
// 	coinKeeper := test.TA().GetCoinKeeper()
// 	// TODO: make the concurrency configurable
// 	orderKeeper := test.TA().GetOrderKeeper()

// 	ctx := test.TA().NewContext(true, abci.Header{})
// 	acct, _ := sdk.AccAddressFromHex("bc1handlenew0rder2")
// 	msg := o.NewOrderMsg(acct, "Reject1", "BTC_BNB", 2, 1, 100, 200, 1)

// 	res := o.handleNewOrder(ctx, orderKeeper, accountMapper, msg)
// 	t.Log(res)
// }
