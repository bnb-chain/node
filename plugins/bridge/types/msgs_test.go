package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mock"
)

func TestTransferMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})

	nonEmptyEthAddr := EthereumAddress(common.BytesToAddress([]byte{1}))
	emptyEthAddr := EthereumAddress(common.BytesToAddress([]byte{0}))

	tests := []struct {
		transferMsg  TransferMsg
		expectedPass bool
	}{
		{
			NewTransferMsg("test", 1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, sdk.ValAddress(addrs[0])),
			true,
		}, {
			NewTransferMsg("", 1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, sdk.ValAddress(addrs[0])),
			false,
		}, {
			NewTransferMsg("test", -1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, sdk.ValAddress(addrs[0])),
			false,
		}, {
			NewTransferMsg("test", 1, emptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, sdk.ValAddress(addrs[0])),
			false,
		}, {
			NewTransferMsg("test", 1, nonEmptyEthAddr, emptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, sdk.ValAddress(addrs[0])),
			false,
		}, {
			NewTransferMsg("test", 1, nonEmptyEthAddr, nonEmptyEthAddr, sdk.AccAddress{},
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, sdk.ValAddress(addrs[0])),
			false,
		}, {
			NewTransferMsg("test", 1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 0}, sdk.Coin{Denom: "BNB", Amount: 2}, sdk.ValAddress(addrs[0])),
			false,
		}, {
			NewTransferMsg("test", 1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 0}, sdk.ValAddress(addrs[0])),
			false,
		}, {
			NewTransferMsg("test", 1, nonEmptyEthAddr, nonEmptyEthAddr, addrs[0],
				sdk.Coin{Denom: "BNB", Amount: 2}, sdk.Coin{Denom: "BNB", Amount: 2}, sdk.ValAddress{}),
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
