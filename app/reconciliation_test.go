package app

import (
	"math"
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func Test_Reconciliation_Overflow(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should panic for overflow")
		}
	}()

	accountPre := types.Coins{
		types.NewCoin("BNB", 10),
	}
	accountCurrent := types.Coins{
		types.NewCoin("BNB", math.MaxInt64),
	}
	tokenPre := types.Coins{
		types.NewCoin("BNB", 10),
	}
	tokenCurrent := types.Coins{
		types.NewCoin("BNB", math.MaxInt64),
	}

	_ = accountPre.Plus(tokenCurrent).IsEqual(accountCurrent.Plus(tokenPre))
}

func Test_Reconciliation_NoOverflow(t *testing.T) {
	accountPre := types.Coins{
		types.NewCoin("BNB", 10),
	}
	accountCurrent := types.Coins{
		types.NewCoin("BNB", math.MaxInt64),
	}
	tokenPre := types.Coins{
		types.NewCoin("BNB", 10),
	}
	tokenCurrent := types.Coins{
		types.NewCoin("BNB", math.MaxInt64),
	}

	equal := accountCurrent.Plus(accountPre.Negative()).IsEqual(tokenCurrent.Plus(tokenPre.Negative()))
	require.True(t, equal)

	accountPre = types.Coins{
		types.NewCoin("BNB", math.MaxInt64),
	}
	accountCurrent = types.Coins{
		types.NewCoin("BNB", 10),
	}
	tokenPre = types.Coins{
		types.NewCoin("BNB", math.MaxInt64),
	}
	tokenCurrent = types.Coins{
		types.NewCoin("BNB", 10),
	}

	equal = accountCurrent.Plus(accountPre.Negative()).IsEqual(tokenCurrent.Plus(tokenPre.Negative()))
	require.True(t, equal)
}
