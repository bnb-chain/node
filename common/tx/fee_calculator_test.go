package tx_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
)

func TestFixedFeeCalculator(t *testing.T) {
	_, addr := privAndAddr()
	msg := sdk.NewTestMsg(addr)

	calculator := tx.FixedFeeCalculator(10, types.FeeFree)
	fee := calculator(msg)
	require.Equal(t, types.FeeFree, fee.Type)
	require.Equal(t, sdk.Coins{}, fee.Tokens)

	calculator = tx.FixedFeeCalculator(10, types.FeeForAll)
	fee = calculator(msg)
	require.Equal(t, types.FeeForAll, fee.Type)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeToken, 10)}, fee.Tokens)

	calculator = tx.FixedFeeCalculator(10, types.FeeForProposer)
	fee = calculator(msg)
	require.Equal(t, types.FeeForProposer, fee.Type)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeToken, 10)}, fee.Tokens)
}

func TestFreeFeeCalculator(t *testing.T) {
	_, addr := privAndAddr()
	msg := sdk.NewTestMsg(addr)

	calculator := tx.FreeFeeCalculator()
	fee := calculator(msg)
	require.Equal(t, types.FeeFree, fee.Type)
	require.Equal(t, sdk.Coins{}, fee.Tokens)
}

func TestRegisterAndGetCalculators(t *testing.T) {
	_, addr := privAndAddr()
	msg := sdk.NewTestMsg(addr)

	tx.RegisterCalculator(msg.Type(), tx.FixedFeeCalculator(10, types.FeeForProposer))
	calculator := tx.GetCalculator(msg.Type())
	require.NotNil(t, calculator)
	fee := calculator(msg)
	require.Equal(t, types.FeeForProposer, fee.Type)
	require.Equal(t, sdk.Coins{sdk.NewCoin(types.NativeToken, 10)}, fee.Tokens)

	tx.UnsetAllCalculator()
	require.Nil(t, tx.GetCalculator(msg.Type()))
}
