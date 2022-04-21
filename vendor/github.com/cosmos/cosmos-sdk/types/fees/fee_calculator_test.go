package fees

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/cosmos/cosmos-sdk/types"
)

func TestFixedFeeCalculator(t *testing.T) {
	_, addr := privAndAddr()
	msg := types.NewTestMsg(addr)
	calculator := FixedFeeCalculator(10, types.FeeFree)
	fee := calculator(msg)
	require.Equal(t, types.FeeFree, fee.Type)
	require.Equal(t, types.Coins{}, fee.Tokens)

	calculator = FixedFeeCalculator(10, types.FeeForAll)
	fee = calculator(msg)
	require.Equal(t, types.FeeForAll, fee.Type)
	require.Equal(t, types.Coins{types.NewCoin(types.NativeTokenSymbol, 10)}, fee.Tokens)

	calculator = FixedFeeCalculator(10, types.FeeForProposer)
	fee = calculator(msg)
	require.Equal(t, types.FeeForProposer, fee.Type)
	require.Equal(t, types.Coins{types.NewCoin(types.NativeTokenSymbol, 10)}, fee.Tokens)
}

func TestFreeFeeCalculator(t *testing.T) {
	_, addr := privAndAddr()
	msg := types.NewTestMsg(addr)

	calculator := FreeFeeCalculator()
	fee := calculator(msg)
	require.Equal(t, types.FeeFree, fee.Type)
	require.Equal(t, types.Coins{}, fee.Tokens)
}

func TestRegisterAndGetCalculators(t *testing.T) {
	_, addr := privAndAddr()
	msg := types.NewTestMsg(addr)

	RegisterCalculator(msg.Type(), FixedFeeCalculator(10, types.FeeForProposer))
	calculator := GetCalculator(msg.Type())
	require.NotNil(t, calculator)
	fee := calculator(msg)
	require.Equal(t, types.FeeForProposer, fee.Type)
	require.Equal(t, types.Coins{types.NewCoin(types.NativeTokenSymbol, 10)}, fee.Tokens)

	UnsetAllCalculators()
	require.Nil(t, GetCalculator(msg.Type()))
}

func privAndAddr() (crypto.PrivKey, types.AccAddress) {
	priv := secp256k1.GenPrivKey()
	addr := types.AccAddress(priv.PubKey().Address())
	return priv, addr
}
