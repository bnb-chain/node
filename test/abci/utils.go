package abci

import (
	"os"

	"github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app"
)

type TestClient struct {
	cl  abcicli.Client
	cdc *wire.Codec
}

func NewTestClient(a *app.BinanceChain) *TestClient {
	a.SetAnteHandler(nil) // clear AnteHandler to skip the signature verification step
	return &TestClient{abcicli.NewLocalClient(nil, a), app.MakeCodec()}
}

func (tc *TestClient) DeliverTxAsync(msg sdk.Msg, cdc *wire.Codec) *abcicli.ReqRes {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, auth.NewStdFee(0), nil, "test")
	tx, _ := tc.cdc.MarshalBinary(stdtx)
	return tc.cl.DeliverTxAsync(tx)
}

func (tc *TestClient) CheckTxAsync(msg sdk.Msg, cdc *wire.Codec) *abcicli.ReqRes {
	return nil
}

func (tc *TestClient) DeliverTxSync(msg sdk.Msg, cdc *wire.Codec) (*types.ResponseDeliverTx, error) {
	return nil, nil
}

func (tc *TestClient) CheckTxSync(msg sdk.Msg, cdc *wire.Codec) (*types.ResponseCheckTx, error) {
	return nil, nil
}

// util objects
var (
	memDB   = db.NewMemDB()
	logger  = log.NewTMLogger(os.Stdout)
	testApp = app.NewBinanceChain(logger, memDB, os.Stdout)
	tc      = NewTestClient(testApp)
)

func TC() *TestClient {
	return tc
}
