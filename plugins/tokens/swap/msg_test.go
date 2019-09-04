package swap

import (
	"encoding/hex"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mock"

	"github.com/stretchr/testify/require"
)

func TestHTLTMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(2, sdk.Coins{})
	tests := []struct {
		From                sdk.AccAddress
		To                  sdk.AccAddress
		RecipientOtherChain string
		SenderOtherChain    string
		RandomNumberHash    string
		Timestamp           int64
		OutAmount           sdk.Coins
		ExpectedIncome      string
		HeightSpan          int64
		Pass                bool
		CrossChain          bool
		ErrorCode           sdk.CodeType
	}{
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "491e71b619878c083eaf2894718383c7eb15eb17",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "10000:BNB",
			HeightSpan:          1000,
			Pass:                true,
			CrossChain:          true,
			ErrorCode:           0,
		},
		{
			From:                addrs[0][1:],
			To:                  addrs[1],
			RecipientOtherChain: "491e71b619878c083eaf2894718383c7eb15eb17",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "10000:BNB",
			HeightSpan:          1000,
			Pass:                false,
			CrossChain:          true,
			ErrorCode:           CodeClaimExpiredSwap,
		},
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "491e71b619878c083eaf2894718383c7eb15eb17491e71b619878c083eaf2894718383c7eb15eb17491e71b619878c083eaf2894718383c7eb15eb17491e71b619878c083eaf2894718383c7eb15eb17",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "10000:BNB",
			HeightSpan:          1000,
			Pass:                false,
			CrossChain:          true,
			ErrorCode:           CodeInvalidAddrOtherChain,
		},
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "491e71b619878c083eaf2894718383c7eb15eb17",
			RandomNumberHash:    "54be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "10000:BNB",
			HeightSpan:          1000,
			Pass:                false,
			CrossChain:          true,
			ErrorCode:           CodeInvalidRandomNumberHash,
		},
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "491e71b619878c083eaf2894718383c7eb15eb17",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", -10000}},
			ExpectedIncome:      "10000:BNB",
			HeightSpan:          1000,
			Pass:                false,
			CrossChain:          true,
			ErrorCode:           sdk.CodeInvalidCoins,
		},
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "491e71b619878c083eaf2894718383c7eb15eb17",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "10000:BNB",
			HeightSpan:          100,
			Pass:                false,
			CrossChain:          true,
			ErrorCode:           CodeInvalidHeightSpan,
		},
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "491e71b619878c083eaf2894718383c7eb15eb17",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "10000:BNB",
			HeightSpan:          1000000,
			Pass:                false,
			CrossChain:          true,
			ErrorCode:           CodeInvalidHeightSpan,
		},
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "10000:BNB",
			HeightSpan:          1000,
			CrossChain:          true,
			Pass:                false,
			ErrorCode:           CodeInvalidAddrOtherChain,
		},
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "10000:BNB",
			HeightSpan:          1000,
			CrossChain:          false,
			Pass:                true,
			ErrorCode:           0,
		},
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "491e71b619878c083eaf2894718383c7eb15eb17",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "1000000000000000000000000000000000000000000000000000000000000:BNB",
			HeightSpan:          1000,
			CrossChain:          true,
			Pass:                false,
			ErrorCode:           CodeInvalidExpectedIncome,
		},
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "491e71b619878c083eaf2894718383c7eb15eb17",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "1000BNB",
			HeightSpan:          1000,
			CrossChain:          true,
			Pass:                false,
			ErrorCode:           CodeInvalidExpectedIncome,
		},
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "491e71b619878c083eaf2894718383c7eb15eb17",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "-1000:BNB",
			HeightSpan:          1000,
			CrossChain:          true,
			Pass:                false,
			ErrorCode:           CodeInvalidExpectedIncome,
		},
	}

	for i, tc := range tests {
		randomNumberHash, _ := hex.DecodeString(tc.RandomNumberHash)
		msg := NewHTLTMsg(tc.From, tc.To, tc.RecipientOtherChain, tc.SenderOtherChain, randomNumberHash, tc.Timestamp, tc.OutAmount, tc.ExpectedIncome, tc.HeightSpan, tc.CrossChain)

		err := msg.ValidateBasic()
		if tc.Pass {
			require.Nil(t, err, "test: %v", i)
		} else {
			require.NotNil(t, err, "test: %v", i)
			require.Equal(t, err.Code(), tc.ErrorCode)
		}
	}
}

func TestDepositHTLTMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(2, sdk.Coins{})
	tests := []struct {
		From      sdk.AccAddress
		SwapID    string
		OutAmount sdk.Coins
		Pass      bool
		ErrorCode sdk.CodeType
	}{
		{
			From:      addrs[0],
			OutAmount: sdk.Coins{sdk.Coin{"BNB", 10000}},
			SwapID:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:      true,
			ErrorCode: 0,
		},
		{
			From:      addrs[0][1:],
			OutAmount: sdk.Coins{sdk.Coin{"BNB", 10000}},
			SwapID:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:      false,
			ErrorCode: CodeClaimExpiredSwap,
		},
		{
			From:      addrs[0],
			OutAmount: sdk.Coins{sdk.Coin{"BNB", 0}},
			SwapID:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:      false,
			ErrorCode: sdk.CodeInvalidCoins,
		},
		{
			From:      addrs[0],
			OutAmount: sdk.Coins{sdk.Coin{"BNB", 10000}},
			SwapID:    "543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:      false,
			ErrorCode: CodeInvalidSwapID,
		},
	}

	for i, tc := range tests {
		swapID, _ := hex.DecodeString(tc.SwapID)
		msg := NewDepositHTLTMsg(tc.From, tc.OutAmount, swapID)

		err := msg.ValidateBasic()
		if tc.Pass {
			require.Nil(t, err, "test: %v", i)
		} else {
			require.NotNil(t, err, "test: %v", i)
			require.Equal(t, tc.ErrorCode, err.Code())
		}
	}
}

func TestClaimHTLTMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(2, sdk.Coins{})
	tests := []struct {
		From         sdk.AccAddress
		SwapID       string
		RandomNumber string
		Pass         bool
		ErrorCode    sdk.CodeType
	}{
		{
			From:         addrs[0],
			RandomNumber: "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649",
			SwapID:       "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:         true,
			ErrorCode:    0,
		},
		{
			From:         addrs[0][1:],
			RandomNumber: "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649",
			SwapID:       "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:         false,
			ErrorCode:    CodeClaimExpiredSwap,
		},
		{
			From:         addrs[0],
			RandomNumber: "fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649",
			SwapID:       "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:         false,
			ErrorCode:    CodeInvalidRandomNumber,
		},
		{
			From:         addrs[0],
			RandomNumber: "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649",
			SwapID:       "543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:         false,
			ErrorCode:    CodeInvalidSwapID,
		},
	}

	for i, tc := range tests {
		randomNumber, _ := hex.DecodeString(tc.RandomNumber)
		swapID, _ := hex.DecodeString(tc.SwapID)
		msg := NewClaimHTLTMsg(tc.From, swapID, randomNumber)

		err := msg.ValidateBasic()
		if tc.Pass {
			require.Nil(t, err, "test: %v", i)
		} else {
			require.NotNil(t, err, "test: %v", i)
			require.Equal(t, tc.ErrorCode, err.Code())
		}
	}
}

func TestRefundHTLTMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(2, sdk.Coins{})
	tests := []struct {
		From      sdk.AccAddress
		SwapID    string
		Pass      bool
		ErrorCode sdk.CodeType
	}{
		{
			From:      addrs[0],
			SwapID:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:      true,
			ErrorCode: 0,
		},
		{
			From:      addrs[0][2:],
			SwapID:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:      false,
			ErrorCode: CodeClaimExpiredSwap,
		},
		{
			From:      addrs[0],
			SwapID:    "543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:      false,
			ErrorCode: CodeInvalidSwapID,
		},
	}

	for i, tc := range tests {
		swapID, _ := hex.DecodeString(tc.SwapID)
		msg := NewRefundHTLTMsg(tc.From, swapID)

		err := msg.ValidateBasic()
		if tc.Pass {
			require.Nil(t, err, "test: %v", i)
		} else {
			require.NotNil(t, err, "test: %v", i)
			require.Equal(t, tc.ErrorCode, err.Code())
		}
	}
}
