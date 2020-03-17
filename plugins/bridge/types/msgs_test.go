package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestTransferMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})

	nonEmptyEthAddr := EthereumAddress(common.BytesToAddress([]byte{1}))
	emptyEthAddr := EthereumAddress(common.BytesToAddress([]byte{0}))

	tests := []struct {
		transferMsg  TransferInMsg
		expectedPass bool
	}{
		{
			NewTransferInMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			true,
		},
		{
			NewTransferInMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 0),
			false,
		}, {
			NewTransferInMsg(-1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			false,
		}, {
			NewTransferInMsg(1, emptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			false,
		}, {
			NewTransferInMsg(1, nonEmptyEthAddr, emptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			false,
		}, {
			NewTransferInMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, sdk.AccAddress{},
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			false,
		}, {
			NewTransferInMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 0}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			false,
		}, {
			NewTransferInMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 0}, addrs[0], 1000),
			false,
		}, {
			NewTransferInMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, sdk.AccAddress{}, 1000),
			false,
		},
	}

	for i, test := range tests {
		if test.expectedPass {
			require.Nil(t, test.transferMsg.ValidateBasic(), "test: %v", i)
		} else {
			require.NotNil(t, test.transferMsg.ValidateBasic(), "test: %v", i)
		}
	}
}

func TestTimeoutMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})

	tests := []struct {
		timeoutMsg   UpdateTransferOutMsg
		expectedPass bool
	}{
		{
			NewUpdateTransferOutMsg(addrs[0], 1, sdk.Coin{"BNB", 10}, addrs[0]),
			true,
		}, {
			NewUpdateTransferOutMsg(sdk.AccAddress{1}, 1, sdk.Coin{"BNB", 10}, addrs[0]),
			false,
		}, {
			NewUpdateTransferOutMsg(addrs[0], -1, sdk.Coin{"BNB", 10}, addrs[0]),
			false,
		}, {
			NewUpdateTransferOutMsg(addrs[0], 1, sdk.Coin{"BNB", 0}, addrs[0]),
			false,
		}, {
			NewUpdateTransferOutMsg(addrs[0], 1, sdk.Coin{"BNB", 10}, sdk.AccAddress{1, 2}),
			true,
		},
	}

	for i, test := range tests {
		if test.expectedPass {
			require.Nil(t, test.timeoutMsg.ValidateBasic(), "test: %v", i)
		} else {
			require.NotNil(t, test.timeoutMsg.ValidateBasic(), "test: %v", i)
		}
	}
}

func TestBindMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})

	nonEmptyEthAddr := EthereumAddress(common.BytesToAddress([]byte{1}))
	emptyEthAddr := EthereumAddress(common.BytesToAddress([]byte{0}))

	tests := []struct {
		bindMsg      BindMsg
		expectedPass bool
	}{
		{
			NewBindMsg(addrs[0], "BNB", 1, nonEmptyEthAddr, 1, 100),
			true,
		}, {
			NewBindMsg(addrs[0], "", 1, nonEmptyEthAddr, 1, 100),
			false,
		}, {
			NewBindMsg(addrs[0], "BNB", 0, nonEmptyEthAddr, 1, 100),
			false,
		}, {
			NewBindMsg(sdk.AccAddress{0, 1}, "BNB", 1, nonEmptyEthAddr, 1, 100),
			false,
		}, {
			NewBindMsg(addrs[0], "BNB", 1, emptyEthAddr, 1, 100),
			false,
		}, {
			NewBindMsg(addrs[0], "BNB", 1, nonEmptyEthAddr, -1, 100),
			false,
		}, {
			NewBindMsg(addrs[0], "BNB", 1, nonEmptyEthAddr, 20, 100),
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
