package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/binance-chain/node/common/testutils"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/tokens/client/rest"
)

func TestAccount_ToBalances(t *testing.T) {
	privKey, addr := testutils.PrivAndAddr()
	acc := &types.AppAccount{
		BaseAccount: auth.BaseAccount{
			Address: addr,
			Coins: nil,
			PubKey: privKey.PubKey(),
			AccountNumber: 1,
			Sequence :1,
		},
	}
	balances := toTokenBalances(acc)
	require.Equal(t, []rest.TokenBalance{}, balances)
	acc.SetCoins(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10e8)})
	balances = toTokenBalances(acc)
	require.Equal(t, 1, len(balances))
	require.Equal(t, int64(10e8), balances[0].Free.ToInt64())
	require.Equal(t, int64(0), balances[0].Locked.ToInt64())
	require.Equal(t, int64(0), balances[0].Frozen.ToInt64())

	acc.SetCoins(sdk.Coins{})
	acc.SetLockedCoins(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e8)})
	balances = toTokenBalances(acc)
	require.Equal(t, 1, len(balances))
	require.Equal(t, int64(0), balances[0].Free.ToInt64())
	require.Equal(t, int64(1e8), balances[0].Locked.ToInt64())
	require.Equal(t, int64(0), balances[0].Frozen.ToInt64())

	acc.SetFrozenCoins(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 1e8)})
	balances = toTokenBalances(acc)
	require.Equal(t, 1, len(balances))
	require.Equal(t, int64(0), balances[0].Free.ToInt64())
	require.Equal(t, int64(1e8), balances[0].Locked.ToInt64())
	require.Equal(t, int64(1e8), balances[0].Frozen.ToInt64())
}
