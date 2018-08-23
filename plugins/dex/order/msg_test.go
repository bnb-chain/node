package order

import (
	"regexp"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/tendermint/tendermint/libs/bech32"
	rpcclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/BiJie/BinanceChain/common"
)

func newCoreContext(seq int64) context.CoreContext {
	// TODO: necessary to make a CoreContext, maybe revisit
	nodeURI := "tcp://localhost:26657"
	rpc := rpcclient.NewHTTP(nodeURI, "/")
	return context.CoreContext{
		Client:          rpc,
		NodeURI:         nodeURI,
		AccountStore:    cmn.AccountStoreName,
		Sequence:        seq,
		FromAddressName: "me",
	}
}

func TestIsValidSide(t *testing.T) {
	assert := assert.New(t)
	assert.True(IsValidSide(1))
	assert.True(IsValidSide(2))
	assert.False(IsValidSide(0))
	assert.False(IsValidSide(3))
}

func TestIsValidOrderType(t *testing.T) {
	assert := assert.New(t)
	assert.False(IsValidOrderType(1))
	assert.True(IsValidOrderType(2))
	assert.False(IsValidOrderType(0))
	assert.False(IsValidOrderType(3))
}

func TestIsValidTimeInForce(t *testing.T) {
	assert := assert.New(t)
	assert.True(IsValidTimeInForce(1))
	assert.False(IsValidTimeInForce(2))
	assert.False(IsValidTimeInForce(0))
	assert.True(IsValidTimeInForce(3))
}

func TestNewOrderMsg_ValidateBasic(t *testing.T) {
	assert := assert.New(t)
	add, e := bech32.ConvertAndEncode(sdk.Bech32PrefixAccAddr, []byte("NEWORDERVALIDATE"))
	acct, e := sdk.AccAddressFromBech32(add)
	t.Log(e)
	msg := NewNewOrderMsg(acct, "addr-1", 1, "BTC.B_BNB", 355, 100)
	assert.Nil(msg.ValidateBasic())
	msg = NewNewOrderMsg(acct, "addr-1", 5, "BTC.B_BNB", 355, 100)
	assert.Regexp(regexp.MustCompile(".*Invalid side:5.*"), msg.ValidateBasic().Error())
	msg = NewNewOrderMsg(acct, "addr-1", 2, "BTC.B_BNB", -355, 100)
	assert.Regexp(regexp.MustCompile(".*Zero/Negative Number.*"), msg.ValidateBasic().Error())
	msg = NewNewOrderMsg(acct, "addr-1", 2, "BTC.B_BNB", 355, 0)
	assert.Regexp(regexp.MustCompile(".*Zero/Negative Number.*"), msg.ValidateBasic().Error())
	msg = NewNewOrderMsg(acct, "addr-1", 2, "BTC.B_BNB", 355, 10)
	msg.TimeInForce = 5
	assert.Regexp(regexp.MustCompile(".*Invalid TimeInForce.*"), msg.ValidateBasic().Error())
}

func TestCancelOrderMsg_ValidateBasic(t *testing.T) {
	assert := assert.New(t)
	msg := NewCancelOrderMsg(sdk.AccAddress{}, "order3", "order1")
	assert.NotNil(msg.ValidateBasic())
}

func TestGenerateOrderId(t *testing.T) {
	cctx := newCoreContext(5)
	viper.SetDefault(client.FlagSequence, "5")

	saddr := "cosmosaccaddr1atcjghcs273lg95p2kcrn509gdyx2h2g83l0mj"
	addr, err := sdk.AccAddressFromBech32(saddr)
	if err != nil {
		panic(err)
	}

	cctx, err = context.EnsureSequence(cctx)
	if err != nil {
		panic(err)
	}

	orderID := GenerateOrderID(cctx.Sequence, addr)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "cosmosaccaddr1atcjghcs273lg95p2kcrn509gdyx2h2g83l0mj-5", orderID)
}
