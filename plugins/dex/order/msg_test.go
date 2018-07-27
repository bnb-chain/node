package order

import (
	"regexp"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/bech32"
)

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

func TestValidateSymbol(t *testing.T) {
	assert := assert.New(t)
	assert.Nil(ValidateSymbol("BTC.B_BNB"))
	assert.NotNil(ValidateSymbol("BNB"))
	assert.NotNil(ValidateSymbol("_BNB"))
	assert.NotNil(ValidateSymbol("BNB_"))
}

func TestNewOrderMsg_ValidateBasic(t *testing.T) {
	assert := assert.New(t)
	add, e := bech32.ConvertAndEncode(sdk.Bech32PrefixAccAddr, []byte("NEWORDERVALIDATE"))
	acct, e := sdk.AccAddressFromBech32(add)
	t.Log(e)
	msg := NewNewOrderMsg(acct, "order1", 1, "BTC.B_BNB", 355, 100)
	assert.Nil(msg.ValidateBasic())
	msg = NewNewOrderMsg(acct, "order1", 5, "BTC.B_BNB", 355, 100)
	assert.Regexp(regexp.MustCompile(".*Invalid side:5.*"), msg.ValidateBasic().Error())
	msg = NewNewOrderMsg(acct, "order1", 2, "BTC.B_BNB", -355, 100)
	assert.Regexp(regexp.MustCompile(".*Zero/Negative Number.*"), msg.ValidateBasic().Error())
	msg = NewNewOrderMsg(acct, "order1", 2, "BTC.B_BNB", 355, 0)
	assert.Regexp(regexp.MustCompile(".*Zero/Negative Number.*"), msg.ValidateBasic().Error())
	msg = NewNewOrderMsg(acct, "order1", 2, "BTC.BBNB", 355, 100)
	assert.Regexp(regexp.MustCompile(".*Invalid trade symbol.*"), msg.ValidateBasic().Error())
	msg = NewNewOrderMsg(acct, "order1", 2, "BTC.B_BNB", 355, 10)
	msg.TimeInForce = 5
	assert.Regexp(regexp.MustCompile(".*Invalid TimeInForce.*"), msg.ValidateBasic().Error())
}

func TestCancelOrderMsg_ValidateBasic(t *testing.T) {
	assert := assert.New(t)
	msg := NewCancelOrderMsg(sdk.AccAddress{}, "order3", "order1")
	assert.NotNil(msg.ValidateBasic())
}
