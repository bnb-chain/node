package tokens_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	bca "github.com/BiJie/BinanceChain/app"
	common "github.com/BiJie/BinanceChain/common/types"
)

// util objects
var (
	db     = dbm.NewMemDB()
	logger = log.NewTMLogger(os.Stdout)
	app    = bca.NewBinanceChain(logger, db, os.Stdout)
)

func Test_Tokens_ABCI_GetInfo_Success(t *testing.T) {
	path := "tokens/info/XXX" // XXX created below

	pk := ed25519.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pk.Address())

	ctx := app.NewContext(true, abci.Header{})
	token := common.NewToken("XXX", "XXX", 10000000000, addr)
	err := app.TokenMapper.NewToken(ctx, token)
	if err != nil {
		t.Fatal(err.Error())
	}

	query := abci.RequestQuery{
		Path:   path,
		Data:   []byte(""),
		Height: 100,
	}
	res := app.Query(query)

	var actual common.Token
	cdc := app.GetCodec()
	err = cdc.UnmarshalBinary(res.Value, &actual)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.True(t, sdk.ABCICodeType(res.Code).IsOK())
	assert.Equal(t, token, actual)
}

func Test_Tokens_ABCI_GetInfo_NotFound(t *testing.T) {
	path := "tokens/info/XXY" // will not exist!

	pk := ed25519.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pk.Address())

	ctx := app.NewContext(true, abci.Header{})
	token := common.NewToken("XXX", "XXX", 10000000000, addr)
	err := app.TokenMapper.NewToken(ctx, token)
	if err != nil {
		t.Fatal(err.Error())
	}

	query := abci.RequestQuery{
		Path:   path,
		Data:   []byte(""),
		Height: 100,
	}
	res := app.Query(query)

	assert.False(t, sdk.ABCICodeType(res.Code).IsOK())
}

func Test_Tokens_ABCI_GetInfo_EmptySymbol(t *testing.T) {
	path := "tokens/info/" // blank symbol param!

	query := abci.RequestQuery{
		Path:   path, // does not exist!
		Data:   []byte(""),
		Height: 100,
	}
	res := app.Query(query)

	assert.False(t, sdk.ABCICodeType(res.Code).IsOK())
	assert.Equal(t, "empty symbol not permitted", res.GetLog())
}
