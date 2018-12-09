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
	pk     = ed25519.GenPrivKey().PubKey()
	addr   = sdk.AccAddress(pk.Address())
	token1Ptr, _ = common.NewToken("XXX", "XXX-000", 10000000000, addr)
	token2Ptr, _ = common.NewToken("XXY", "XXY-000", 10000000000, addr)
	token1 = *token1Ptr
	token2 = *token2Ptr
)

func Test_Tokens_ABCI_GetInfo_Success(t *testing.T) {
	path := "/tokens/info/XXX-000" // XXX created below

	ctx := app.NewContext(sdk.RunTxModeCheck, abci.Header{})
	err := app.TokenMapper.NewToken(ctx, token1)
	if err != nil {
		t.Fatal(err.Error())
	}

	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	res := app.Query(query)

	var actual common.Token
	cdc := app.GetCodec()
	err = cdc.UnmarshalBinary(res.Value, &actual)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.True(t, sdk.ABCICodeType(res.Code).IsOK())
	assert.Equal(t, token1, actual)
}

func Test_Tokens_ABCI_GetInfo_Error_NotFound(t *testing.T) {
	path := "/tokens/info/XXY-000" // will not exist!

	ctx := app.NewContext(sdk.RunTxModeCheck, abci.Header{})
	err := app.TokenMapper.NewToken(ctx, token1)
	if err != nil {
		t.Fatal(err.Error())
	}

	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	res := app.Query(query)

	assert.False(t, sdk.ABCICodeType(res.Code).IsOK())
}

func Test_Tokens_ABCI_GetInfo_Error_EmptySymbol(t *testing.T) {
	path := "/tokens/info/" // blank symbol param!

	query := abci.RequestQuery{
		Path: path, // does not exist!
		Data: []byte(""),
	}
	res := app.Query(query)

	assert.False(t, sdk.ABCICodeType(res.Code).IsOK())
	assert.Equal(t, "empty symbol not permitted", res.GetLog())
}

func Test_Tokens_ABCI_GetTokens_Success(t *testing.T) {
	path := "/tokens/list/0/5"

	ctx := app.NewContext(sdk.RunTxModeCheck, abci.Header{})
	err := app.TokenMapper.NewToken(ctx, token1)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = app.TokenMapper.NewToken(ctx, token2)
	if err != nil {
		t.Fatal(err.Error())
	}

	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	res := app.Query(query)

	cdc := app.GetCodec()
	actual := make([]common.Token, 2)
	err = cdc.UnmarshalBinary(res.Value, &actual)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.True(t, sdk.ABCICodeType(res.Code).IsOK())
	assert.Equal(t, []common.Token{
		token1, token2,
	}, actual)
}

func Test_Tokens_ABCI_GetTokens_Success_WithOffset(t *testing.T) {
	path := "/tokens/list/1/5"

	ctx := app.NewContext(sdk.RunTxModeCheck, abci.Header{})
	err := app.TokenMapper.NewToken(ctx, token1)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = app.TokenMapper.NewToken(ctx, token2)
	if err != nil {
		t.Fatal(err.Error())
	}

	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	res := app.Query(query)

	cdc := app.GetCodec()
	actual := make([]common.Token, 1)
	err = cdc.UnmarshalBinary(res.Value, &actual)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.True(t, sdk.ABCICodeType(res.Code).IsOK())
	assert.Equal(t, []common.Token{
		token2,
	}, actual)
}

func Test_Tokens_ABCI_GetTokens_Success_WithLimit(t *testing.T) {
	path := "/tokens/list/0/1"

	ctx := app.NewContext(sdk.RunTxModeCheck, abci.Header{})
	err := app.TokenMapper.NewToken(ctx, token1)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = app.TokenMapper.NewToken(ctx, token2)
	if err != nil {
		t.Fatal(err.Error())
	}

	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	res := app.Query(query)

	cdc := app.GetCodec()
	actual := make([]common.Token, 1)
	err = cdc.UnmarshalBinary(res.Value, &actual)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.True(t, sdk.ABCICodeType(res.Code).IsOK())
	assert.Equal(t, []common.Token{
		token1,
	}, actual)
}

func Test_Tokens_ABCI_GetTokens_Error_ZeroLimit(t *testing.T) {
	path := "/tokens/list/0/0"

	ctx := app.NewContext(sdk.RunTxModeCheck, abci.Header{})
	err := app.TokenMapper.NewToken(ctx, token1)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = app.TokenMapper.NewToken(ctx, token2)
	if err != nil {
		t.Fatal(err.Error())
	}

	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	res := app.Query(query)

	assert.False(t, sdk.ABCICodeType(res.Code).IsOK())
}

func Test_Tokens_ABCI_GetTokens_Error_NegativeLimit(t *testing.T) {
	path := "/tokens/list/0/-1"

	ctx := app.NewContext(sdk.RunTxModeCheck, abci.Header{})
	err := app.TokenMapper.NewToken(ctx, token1)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = app.TokenMapper.NewToken(ctx, token2)
	if err != nil {
		t.Fatal(err.Error())
	}

	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	res := app.Query(query)

	assert.False(t, sdk.ABCICodeType(res.Code).IsOK())
}

func Test_Tokens_ABCI_GetTokens_Error_NegativeOffset(t *testing.T) {
	path := "/tokens/list/-1/0"

	ctx := app.NewContext(sdk.RunTxModeCheck, abci.Header{})
	err := app.TokenMapper.NewToken(ctx, token1)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = app.TokenMapper.NewToken(ctx, token2)
	if err != nil {
		t.Fatal(err.Error())
	}

	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	res := app.Query(query)

	assert.False(t, sdk.ABCICodeType(res.Code).IsOK())
}

func Test_Tokens_ABCI_GetTokens_Error_InvalidLimit(t *testing.T) {
	path := "/tokens/list/0/x"

	ctx := app.NewContext(sdk.RunTxModeCheck, abci.Header{})
	err := app.TokenMapper.NewToken(ctx, token1)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = app.TokenMapper.NewToken(ctx, token2)
	if err != nil {
		t.Fatal(err.Error())
	}

	query := abci.RequestQuery{
		Path: path,
		Data: []byte(""),
	}
	res := app.Query(query)

	assert.False(t, sdk.ABCICodeType(res.Code).IsOK())
}
