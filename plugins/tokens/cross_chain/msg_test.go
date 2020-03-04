package cross_chain

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/binance-chain/node/common/types"
)

func TestBindMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})

	nonEmptyEthAddr := types.EthereumAddress(common.BytesToAddress([]byte{1}))
	emptyEthAddr := types.EthereumAddress(common.BytesToAddress([]byte{0}))

	tests := []struct {
		bindMsg      BindMsg
		expectedPass bool
	}{
		{
			NewBindMsg(addrs[0], nonEmptyEthAddr, 1),
			true,
		}, {
			NewBindMsg(sdk.AccAddress{0, 1}, nonEmptyEthAddr, 1),
			false,
		}, {
			NewBindMsg(addrs[0], emptyEthAddr, 1),
			false,
		}, {
			NewBindMsg(addrs[0], nonEmptyEthAddr, -1),
			false,
		}, {
			NewBindMsg(addrs[0], nonEmptyEthAddr, 20),
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
