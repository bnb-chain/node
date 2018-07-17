package order

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
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
	acct, _ := sdk.AccAddressFromHex("1234123412341234")
	msg := NewNewOrderMsg(acct, "order1", 1, "BTC.B_BNB", 355, 100)
	assert.Nil(msg.ValidateBasic())
}

func TestCancelOrderMsg_ValidateBasic(t *testing.T) {

}
