package abci

import (
	"testing"

	common "github.com/BiJie/BinanceChain/common/types"
	o "github.com/BiJie/BinanceChain/plugins/dex/order"
	abci "github.com/tendermint/tendermint/abci/types"
)

func Test_handleNewOrder(t *testing.T) {

	ctx := TA().NewContext(true, abci.Header{})
	add := Account(0).GetAddress()
	msg := o.NewNewOrderMsg(add, "order1", 1, "BTC_BNB", 355, 100)
	res, e := TC().CheckTxSync(msg, TA().GetCodec())
	t.Logf("Result is %v, error is %v", res, e)
	t.Logf("coins are %v, %v", TA().GetCoinKeeper().GetCoins(ctx, add), TA().GetAccountMapper().GetAccount(ctx, add).(common.NamedAccount).GetLockedCoins())
}
