package tokens_test

import (
	"testing"

	"github.com/BiJie/BinanceChain/common/testutils"
	common "github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/param/types"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/cosmos/cosmos-sdk/x/bank"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func newAddr() sdk.AccAddress {
	_, addr := testutils.PrivAndAddr()
	return addr
}

func checkFee(t *testing.T, fee common.Fee, expected int64) {
	require.Equal(t,
		common.NewFee(sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, expected)}, common.FeeForProposer),
		fee)
}

func TestTransferFeeGen(t *testing.T) {
	params := types.TransferFeeParam{
		FixedFeeParams: types.FixedFeeParams{
			MsgType: bank.MsgSend{}.Type(),
			Fee:1e6,
			FeeFor:common.FeeForProposer,
		},
		MultiTransferFee: 8e5,
		LowerLimitAsMulti: 2,
	}

	calculator := tokens.TransferFeeCalculatorGen(&params)

	// (1 addr, 1 coin) : (1 addr, 1 coin)
	msg := bank.MsgSend{
		Inputs: []bank.Input{bank.NewInput(newAddr(), sdk.Coins{ sdk.NewCoin(common.NativeTokenSymbol, 1000)})},
		Outputs:[]bank.Output{bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000)})},
	}
	checkFee(t, calculator(msg), 1e6)

	// (1 addr, 1 coin) : (2 addr, 1 coin)
	msg = bank.MsgSend{
		Inputs: []bank.Input{
			bank.NewInput(newAddr(), sdk.Coins{ sdk.NewCoin(common.NativeTokenSymbol, 1000)}),
		},
		Outputs:[]bank.Output{
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 500)}),
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 500)}),
		},
	}
	checkFee(t, calculator(msg), 16e5)

	// (2 addr, 1 coin) : (1 addr, 1 coin)
	msg = bank.MsgSend{
		Inputs: []bank.Input{
			bank.NewInput(newAddr(), sdk.Coins{ sdk.NewCoin(common.NativeTokenSymbol, 500)}),
			bank.NewInput(newAddr(), sdk.Coins{ sdk.NewCoin(common.NativeTokenSymbol, 500)}),
		},
		Outputs:[]bank.Output{
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000)}),
		},
	}
	checkFee(t, calculator(msg), 16e5)

	// (1 addr, 2 coin) : (1 addr, 2 coin)
	msg = bank.MsgSend{
		Inputs: []bank.Input{
			bank.NewInput(newAddr(), sdk.Coins{ sdk.NewCoin(common.NativeTokenSymbol, 1000), sdk.NewCoin("XYZ", 1000)}),
		},
		Outputs:[]bank.Output{
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000), sdk.NewCoin("XYZ", 1000)}),
		},
	}
	checkFee(t, calculator(msg), 16e5)

	// (1 addr, 2 coin) : (2 addr, 2 coin)
	msg = bank.MsgSend{
		Inputs: []bank.Input{
			bank.NewInput(newAddr(), sdk.Coins{ sdk.NewCoin(common.NativeTokenSymbol, 1000), sdk.NewCoin("XYZ", 1000)}),
		},
		Outputs:[]bank.Output{
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin(common.NativeTokenSymbol, 1000), sdk.NewCoin("XYZ", 500)}),
			bank.NewOutput(newAddr(), sdk.Coins{sdk.NewCoin("XYZ", 500)}),
		},
	}
	checkFee(t, calculator(msg), 24e5)
}


