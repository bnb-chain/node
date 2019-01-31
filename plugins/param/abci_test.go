package param_test

import (
	"bytes"
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	bca "github.com/binance-chain/node/app"
	"github.com/binance-chain/node/plugins/param"
)

// util objects
var (
	db     = dbm.NewMemDB()
	logger = log.NewTMLogger(os.Stdout)
	app    = bca.NewBinanceChain(logger, db, os.Stdout)
)

func Test_Get_Operate_Fee_OK(t *testing.T) {
	path := "/param/fees"

	ctx := app.NewContext(sdk.RunTxModeCheck, abci.Header{})
	testParam := param.DefaultGenesisState

	app.ParamHub.InitGenesis(ctx, testParam)

	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	res := app.Query(query)

	assert.True(t, sdk.ABCICodeType(res.Code).IsOK())
	output, err := app.GetCodec().MarshalJSON(testParam)
	assert.NoError(t, err)
	bytes.Equal(res.GetValue(), output)
}
