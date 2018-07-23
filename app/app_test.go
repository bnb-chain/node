package app_test

import (
	"os"

	"github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app"
	common "github.com/BiJie/BinanceChain/common/types"
)

type TestClient struct {
	cl  abcicli.Client
	cdc *wire.Codec
}

func (tc *TestClient) DeliverTxAsync(msg sdk.Msg, cdc *wire.Codec) *abcicli.ReqRes {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, auth.NewStdFee(0), nil, "test")
	tx, _ := tc.cdc.MarshalBinary(stdtx)
	return tc.cl.DeliverTxAsync(tx)
}

func (tc *TestClient) CheckTxAsync(msg sdk.Msg, cdc *wire.Codec) *abcicli.ReqRes {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, auth.NewStdFee(0), nil, "test")
	tx, _ := tc.cdc.MarshalBinary(stdtx)
	return tc.cl.CheckTxAsync(tx)
}

func (tc *TestClient) DeliverTxSync(msg sdk.Msg, cdc *wire.Codec) (*types.ResponseDeliverTx, error) {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, auth.NewStdFee(0), nil, "test")
	tx, _ := tc.cdc.MarshalBinary(stdtx)
	return tc.cl.DeliverTxSync(tx)
}

func (tc *TestClient) CheckTxSync(msg sdk.Msg, cdc *wire.Codec) (*types.ResponseCheckTx, error) {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, auth.NewStdFee(0), nil, "test")
	tx, _ := tc.cdc.MarshalBinary(stdtx)
	return tc.cl.CheckTxSync(tx)
}

// util objects
var (
	memDB                             = db.NewMemDB()
	logger                            = log.NewTMLogger(os.Stdout)
	testApp                           = app.NewBinanceChain(logger, memDB, os.Stdout)
	genAccs, addrs, pubKeys, privKeys = mock.CreateGenAccounts(4,
		sdk.Coins{sdk.NewCoin("BNB", 500e8), sdk.NewCoin("BTC", 200e8)})
	tc = NewTestClient(testApp)
)

func TC() *TestClient {
	return tc
}

func TA() *app.BinanceChain {
	return testApp
}

func InitAccounts(ctx sdk.Context, app *app.BinanceChain) {
	for _, acc := range genAccs {
		aacc := &common.AppAccount{BaseAccount: auth.BaseAccount{Address: acc.GetAddress(), Coins: acc.GetCoins()}}
		aacc.BaseAccount.AccountNumber = app.AccountMapper.GetNextAccountNumber(ctx)
		app.AccountMapper.SetAccount(ctx, aacc)
	}
}

func ResetAccounts(ctx sdk.Context, app *app.BinanceChain) {
	for _, acc := range genAccs {
		app.AccountMapper.GetAccount(ctx, acc.GetAddress()).SetCoins(sdk.Coins{sdk.NewCoin("BNB", 500e8), sdk.NewCoin("BTC", 200e8)})
	}
}

func Account(i int) auth.Account {
	return genAccs[i]
}

func Address(i int) sdk.AccAddress {
	return addrs[i]
}

func NewTestClient(a *app.BinanceChain) *TestClient {
	a.SetCheckState(types.Header{})
	a.SetAnteHandler(nil) // clear AnteHandler to skip the signature verification step
	return &TestClient{abcicli.NewLocalClient(nil, a), app.MakeCodec()}
}

func GetAvail(ctx sdk.Context, add sdk.AccAddress, ccy string) int64 {
	return TA().CoinKeeper.GetCoins(ctx, add).AmountOf(ccy).Int64()
}

func GetLocked(ctx sdk.Context, add sdk.AccAddress, ccy string) int64 {
	return TA().AccountMapper.GetAccount(ctx, add).(common.NamedAccount).GetLockedCoins().AmountOf(ccy).Int64()
}
