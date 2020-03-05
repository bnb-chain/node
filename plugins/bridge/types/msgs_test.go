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
			NewTransferMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			true,
		},
		{
			NewTransferMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 0),
			false,
		}, {
			NewTransferMsg(-1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			false,
		}, {
			NewTransferMsg(1, emptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			false,
		}, {
			NewTransferMsg(1, nonEmptyEthAddr, emptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			false,
		}, {
			NewTransferMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, sdk.AccAddress{},
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			false,
		}, {
			NewTransferMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 0}, sdk.Coin{Denom: "BNB", Amount: 2}, addrs[0], 1000),
			false,
		}, {
			NewTransferMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 0}, addrs[0], 1000),
			false,
		}, {
			NewTransferMsg(1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
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
		timeoutMsg   TimeoutMsg
		expectedPass bool
	}{
		{
			NewTimeoutMsg(addrs[0], 1, sdk.Coin{"BNB", 10}, sdk.ValAddress(addrs[0])),
			true,
		}, {
			NewTimeoutMsg(sdk.AccAddress{1}, 1, sdk.Coin{"BNB", 10}, sdk.ValAddress(addrs[0])),
			false,
		}, {
			NewTimeoutMsg(addrs[0], -1, sdk.Coin{"BNB", 10}, sdk.ValAddress(addrs[0])),
			false,
		}, {
			NewTimeoutMsg(addrs[0], 1, sdk.Coin{"BNB", 0}, sdk.ValAddress(addrs[0])),
			false,
		}, {
			NewTimeoutMsg(addrs[0], 1, sdk.Coin{"BNB", 10}, sdk.ValAddress{1, 2}),
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
			NewBindMsg(addrs[0], "BNB", nonEmptyEthAddr, 1),
			true,
		}, {
			NewBindMsg(addrs[0], "", nonEmptyEthAddr, 1),
			false,
		}, {
			NewBindMsg(sdk.AccAddress{0, 1}, "BNB", nonEmptyEthAddr, 1),
			false,
		}, {
			NewBindMsg(addrs[0], "BNB", emptyEthAddr, 1),
			false,
		}, {
			NewBindMsg(addrs[0], "BNB", nonEmptyEthAddr, -1),
			false,
		}, {
			NewBindMsg(addrs[0], "BNB", nonEmptyEthAddr, 20),
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
