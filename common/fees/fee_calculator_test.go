package fees_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/BiJie/BinanceChain/common/fees"
	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/types"
)

func TestFixedFeeCalculator(t *testing.T) {
	_, addr := testutils.PrivAndAddr()
	msg := sdk.NewTestMsg(addr)
	calculator := fees.FixedFeeCalculator(10, types.FeeFree)
	fee := calculator(msg)
	require.Equal(t, types.FeeFree, fee.Type)
	require.Equal(t, sdk.Coins{}, fee.Tokens)

	calculator = fees.FixedFeeCalculator(10, types.FeeForAll)
	fee = calculator(msg)
	require.Equal(t, types.FeeForAll, fee.Type)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, fee.Tokens)

	calculator = fees.FixedFeeCalculator(10, types.FeeForProposer)
	fee = calculator(msg)
	require.Equal(t, types.FeeForProposer, fee.Type)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, fee.Tokens)
}

func TestFreeFeeCalculator(t *testing.T) {
	_, addr := testutils.PrivAndAddr()
	msg := sdk.NewTestMsg(addr)

	calculator := fees.FreeFeeCalculator()
	fee := calculator(msg)
	require.Equal(t, types.FeeFree, fee.Type)
	require.Equal(t, sdk.Coins{}, fee.Tokens)
}

func TestRegisterAndGetCalculators(t *testing.T) {
	_, addr := testutils.PrivAndAddr()
	msg := sdk.NewTestMsg(addr)

	fees.RegisterCalculator(msg.Type(), fees.FixedFeeCalculator(10, types.FeeForProposer))
	calculator := fees.GetCalculator(msg.Type())
	require.NotNil(t, calculator)
	fee := calculator(msg)
	require.Equal(t, types.FeeForProposer, fee.Type)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, fee.Tokens)

	fees.UnsetAllCalculators()
	require.Nil(t, fees.GetCalculator(msg.Type()))
}
