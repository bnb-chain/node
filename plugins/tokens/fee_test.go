package tokens_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/binance-chain/node/common/testutils"
	common "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/plugins/tokens"
)

func newAddr() sdk.AccAddress {
	_, addr := testutils.PrivAndAddr()
	return addr
}

func checkFee(t *testing.T, fee sdk.Fee, expected int64) {
	require.Equal(t,
		sdk.NewFee(sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, expected)}, sdk.FeeForProposer),
		fee)
}

func TestTransferFeeGen(t *testing.T) {
	params := types.TransferFeeParam{
		FixedFeeParams: types.FixedFeeParams{
			MsgType: bank.MsgSend{}.Type(),
			Fee:     1e6,
			FeeFor:  sdk.FeeForProposer,
		},
		MultiTransferFee:  8e5,
		LowerLimitAsMulti: 2,
	}

	calculator := tokens.TransferFeeCalculatorGen(&params)

	// (1 addr, 1 coin) : (1 addr, 1 coin)
	msg := bank.MsgSend{
		Inputs:  []bank.Input{bank.NewInput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000)})},
		Outputs: []bank.Output{bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000)})},
	}
	checkFee(t, calculator(msg), 1e6)

	// (1 addr, 1 coin) : (2 addr, 1 coin)
	msg = bank.MsgSend{
		Inputs: []bank.Input{
			bank.NewInput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000)}),
		},
		Outputs: []bank.Output{
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 500)}),
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 500)}),
		},
	}
	checkFee(t, calculator(msg), 16e5)

	// (2 addr, 1 coin) : (1 addr, 1 coin)
	msg = bank.MsgSend{
		Inputs: []bank.Input{
			bank.NewInput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 500)}),
			bank.NewInput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 500)}),
		},
		Outputs: []bank.Output{
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000)}),
		},
	}
	checkFee(t, calculator(msg), 16e5)

	// (1 addr, 2 coin) : (1 addr, 2 coin)
	msg = bank.MsgSend{
		Inputs: []bank.Input{
			bank.NewInput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000), sdk.NewCoin("XYZ", 1000)}),
		},
		Outputs: []bank.Output{
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000), sdk.NewCoin("XYZ", 1000)}),
		},
	}
	checkFee(t, calculator(msg), 16e5)

	// (1 addr, 2 coin) : (2 addr, 2 coin)
	msg = bank.MsgSend{
		Inputs: []bank.Input{
			bank.NewInput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000), sdk.NewCoin("XYZ", 1000)}),
		},
		Outputs: []bank.Output{
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000), sdk.NewCoin("XYZ", 500)}),
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin("XYZ", 500)}),
		},
	}
	checkFee(t, calculator(msg), 24e5)
}

func TestTransferFeeParams_JsonFormat(t *testing.T) {
	params := types.TransferFeeParam{
		FixedFeeParams: types.FixedFeeParams{
			MsgType: bank.MsgSend{}.Type(),
			Fee:     250000,
			FeeFor:  sdk.FeeForProposer,
		},
		MultiTransferFee:  200000,
		LowerLimitAsMulti: 2,
	}

	bz, err := json.Marshal(params)
	fmt.Println(string(bz))
	require.NoError(t, err)

	except := "{\"fixed_fee_params\":{\"msg_type\":\"send\",\"fee\":250000,\"fee_for\":1},\"multi_transfer_fee\":200000,\"lower_limit_as_multi\":2}"
	require.Equal(t, []byte(except), bz)
}
