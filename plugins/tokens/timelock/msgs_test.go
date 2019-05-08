package timelock

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/stretchr/testify/require"
)

func TestTimeLockMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})
	tests := []struct {
		from        sdk.AccAddress
		description string
		amount      sdk.Coins
		lockTime    int64
		pass        bool
		errorCode   sdk.CodeType
	}{
		{
			from:        []byte("abc"),
			description: "desription",
			amount: sdk.Coins{
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  1000,
			pass:      false,
			errorCode: sdk.CodeInvalidAddress,
		},
		{
			from:        addrs[0],
			description: "",
			amount: sdk.Coins{
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  1000,
			pass:      false,
			errorCode: CodeInvalidDescription,
		},
		{
			from:        addrs[0],
			description: strings.Repeat("d", 129),
			amount: sdk.Coins{
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  1000,
			pass:      false,
			errorCode: CodeInvalidDescription,
		},
		{
			from:        addrs[0],
			description: strings.Repeat("d", 120),
			amount: sdk.Coins{
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  -1,
			pass:      false,
			errorCode: CodeInvalidLockTime,
		},
		{
			from:        addrs[0],
			description: strings.Repeat("d", 120),
			amount: sdk.Coins{
				sdk.NewCoin("ANB", -2000e8),
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  1000,
			pass:      false,
			errorCode: sdk.CodeInvalidCoins,
		},
		{
			from:        addrs[0],
			description: strings.Repeat("d", 120),
			amount: sdk.Coins{
				sdk.NewCoin("ANB", 2000e8),
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  1000,
			pass:      true,
			errorCode: sdk.CodeType(0),
		},
	}

	for i, tc := range tests {
		msg := TimeLockMsg{
			From:        tc.from,
			Description: tc.description,
			Amount:      tc.amount,
			LockTime:    tc.lockTime,
		}

		err := msg.ValidateBasic()
		if tc.pass {
			require.Nil(t, err, "test: %v", i)
		} else {
			require.NotNil(t, err, "test: %v", i)
			require.Equal(t, err.Code(), tc.errorCode)
		}
	}
}

func TestTimeRelockMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})
	tests := []struct {
		from        sdk.AccAddress
		id          int64
		description string
		amount      sdk.Coins
		lockTime    int64
		pass        bool
		errorCode   sdk.CodeType
	}{
		{
			from:        addrs[0],
			id:          0,
			description: "desription",
			amount: sdk.Coins{
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  1000,
			pass:      false,
			errorCode: CodeInvalidTimeLockId,
		},
		{
			from:        []byte("abc"),
			id:          1,
			description: "desription",
			amount: sdk.Coins{
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  1000,
			pass:      false,
			errorCode: sdk.CodeInvalidAddress,
		},
		{
			from:        addrs[0],
			id:          1,
			description: strings.Repeat("d", 129),
			amount: sdk.Coins{
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  1000,
			pass:      false,
			errorCode: CodeInvalidDescription,
		},
		{
			from:        addrs[0],
			id:          1,
			description: strings.Repeat("d", 120),
			amount: sdk.Coins{
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  -1,
			pass:      false,
			errorCode: CodeInvalidLockTime,
		},
		{
			from:        addrs[0],
			id:          1,
			description: strings.Repeat("d", 120),
			amount: sdk.Coins{
				sdk.NewCoin("ANB", -2000e8),
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  1000,
			pass:      false,
			errorCode: sdk.CodeInvalidCoins,
		},
		{
			from:        addrs[0],
			id:          1,
			description: "",
			amount:      sdk.Coins{},
			lockTime:    0,
			pass:        false,
			errorCode:   CodeInvalidRelock,
		},
		{
			from:        addrs[0],
			id:          1,
			description: strings.Repeat("d", 120),
			amount: sdk.Coins{
				sdk.NewCoin("ANB", 2000e8),
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  1000,
			pass:      true,
			errorCode: sdk.CodeType(0),
		},
		{
			from:        addrs[0],
			id:          1,
			description: "",
			amount: sdk.Coins{
				sdk.NewCoin("ANB", 2000e8),
				sdk.NewCoin("BNB", 2000e8),
			},
			lockTime:  1000,
			pass:      true,
			errorCode: sdk.CodeType(0),
		},
	}

	for i, tc := range tests {
		msg := TimeRelockMsg{
			From:        tc.from,
			Id:          tc.id,
			Description: tc.description,
			Amount:      tc.amount,
			LockTime:    tc.lockTime,
		}

		err := msg.ValidateBasic()
		if tc.pass {
			require.Nil(t, err, "test: %v", i)
		} else {
			require.NotNil(t, err, "test: %v", i)
			require.Equal(t, err.Code(), tc.errorCode)
		}
	}
}

func TestTimeUnlockMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})
	tests := []struct {
		from      sdk.AccAddress
		id        int64
		pass      bool
		errorCode sdk.CodeType
	}{
		{
			from:      addrs[0],
			id:        0,
			pass:      false,
			errorCode: CodeInvalidTimeLockId,
		},
		{
			from:      []byte("abc"),
			id:        1,
			pass:      false,
			errorCode: sdk.CodeInvalidAddress,
		},
		{
			from:      addrs[0],
			id:        1,
			pass:      true,
			errorCode: sdk.CodeType(0),
		},
	}

	for i, tc := range tests {
		msg := TimeUnlockMsg{
			From: tc.from,
			Id:   tc.id,
		}

		err := msg.ValidateBasic()
		if tc.pass {
			require.Nil(t, err, "test: %v", i)
		} else {
			require.NotNil(t, err, "test: %v", i)
			require.Equal(t, err.Code(), tc.errorCode)
		}
	}
}
