package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBindMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})

	nonEmptySmartChainAddr := SmartChainAddress(common.BytesToAddress([]byte{1}))
	emptySmartChainAddr := SmartChainAddress(common.BytesToAddress([]byte{0}))

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
