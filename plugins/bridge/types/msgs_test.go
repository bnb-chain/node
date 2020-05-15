package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/stretchr/testify/require"
)

func BytesToAddress(b []byte) SmartChainAddress {
	var a SmartChainAddress
	a.SetBytes(b)
	return a
}
func TestBindMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})

	nonEmptySmartChainAddr := SmartChainAddress(BytesToAddress([]byte{1}))
	emptySmartChainAddr := SmartChainAddress(BytesToAddress([]byte{0}))

	tests := []struct {
		bindMsg      BindMsg
		expectedPass bool
	}{
		{
			NewBindMsg(addrs[0], "BNB", 1, nonEmptySmartChainAddr, 1, 100),
			true,
		}, {
			NewBindMsg(addrs[0], "", 1, nonEmptySmartChainAddr, 1, 100),
			false,
		}, {
			NewBindMsg(addrs[0], "BNB", 0, nonEmptySmartChainAddr, 1, 100),
			false,
		}, {
			NewBindMsg(sdk.AccAddress{0, 1}, "BNB", 1, nonEmptySmartChainAddr, 1, 100),
			false,
		}, {
			NewBindMsg(addrs[0], "BNB", 1, emptySmartChainAddr, 1, 100),
			false,
		}, {
			NewBindMsg(addrs[0], "BNB", 1, nonEmptySmartChainAddr, -1, 100),
			false,
		},
	}

	for i, test := range tests {
		if test.expectedPass {
			require.Nil(t, test.bindMsg.ValidateBasic(), "test: %v", i)
		} else {
			require.NotNil(t, test.bindMsg.ValidateBasic(), "test: %v", i)
		}
	}
}
