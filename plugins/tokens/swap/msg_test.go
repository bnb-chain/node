package swap

import (
	"encoding/hex"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mock"

	"github.com/stretchr/testify/require"
)

func TestHashTimerLockTransferMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(2, sdk.Coins{})
	tests := []struct {
		From                sdk.AccAddress
		To                  sdk.AccAddress
		RecipientOtherChain string
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
			ErrorCode:           0x7,
		},
		{
			From:                addrs[0],
			To:                  addrs[1],
			RecipientOtherChain: "491e71b619878c083eaf2894718383c7eb15eb17491e71b619878c083eaf2894718383c7eb15eb17",
			RandomNumberHash:    "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Timestamp:           1564471835,
			OutAmount:           sdk.Coins{sdk.Coin{"BNB", 10000}},
			ExpectedIncome:      "10000:BNB",
			HeightSpan:          1000,
			Pass:                false,
			CrossChain:          true,
			ErrorCode:           0x1,
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
			ErrorCode:           0x2,
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
			ErrorCode:           0x4,
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
			ErrorCode:           0x5,
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
			ErrorCode:           0x5,
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
			ErrorCode:           0x1,
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
			ErrorCode:           0xf,
		},
	}

	for i, tc := range tests {
		recipientOtherChain, _ := hex.DecodeString(tc.RecipientOtherChain)
		randomNumberHash, _ := hex.DecodeString(tc.RandomNumberHash)
		msg := NewHTLTMsg(tc.From, tc.To, recipientOtherChain, randomNumberHash, tc.Timestamp, tc.OutAmount, tc.ExpectedIncome, tc.HeightSpan, tc.CrossChain)

		err := msg.ValidateBasic()
		if tc.Pass {
			require.Nil(t, err, "test: %v", i)
		} else {
			require.NotNil(t, err, "test: %v", i)
			require.Equal(t, tc.ErrorCode, err.Code())
		}
	}
}

func TestClaimHashTimerLockMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(2, sdk.Coins{})
	tests := []struct {
		From             sdk.AccAddress
		RandomNumberHash string
		RandomNumber     string
		Pass             bool
		ErrorCode        sdk.CodeType
	}{
		{
			From:             addrs[0],
			RandomNumber:     "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649",
			RandomNumberHash: "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:             true,
			ErrorCode:        0,
		},
		{
			From:             addrs[0][1:],
			RandomNumber:     "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649",
			RandomNumberHash: "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:             false,
			ErrorCode:        0x7,
		},
		{
			From:             addrs[0],
			RandomNumber:     "fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649",
			RandomNumberHash: "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:             false,
			ErrorCode:        0x3,
		},
		{
			From:             addrs[0],
			RandomNumber:     "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649",
			RandomNumberHash: "543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:             false,
			ErrorCode:        0x2,
		},
	}

	for i, tc := range tests {
		randomNumber, _ := hex.DecodeString(tc.RandomNumber)
		randomNumberHash, _ := hex.DecodeString(tc.RandomNumberHash)
		msg := NewClaimHTLTMsg(tc.From, randomNumberHash, randomNumber)

		err := msg.ValidateBasic()
		if tc.Pass {
			require.Nil(t, err, "test: %v", i)
		} else {
			require.NotNil(t, err, "test: %v", i)
			require.Equal(t, err.Code(), tc.ErrorCode)
		}
	}
}

func TestRefundLockedAssetMsg(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(2, sdk.Coins{})
	tests := []struct {
		From             sdk.AccAddress
		RandomNumberHash string
		Pass             bool
		ErrorCode        sdk.CodeType
	}{
		{
			From:             addrs[0],
			RandomNumberHash: "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:             true,
			ErrorCode:        0,
		},
		{
			From:             addrs[0][2:],
			RandomNumberHash: "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:             false,
			ErrorCode:        0x7,
		},
		{
			From:             addrs[0],
			RandomNumberHash: "543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167",
			Pass:             false,
			ErrorCode:        0x2,
		},
	}

	for i, tc := range tests {
		randomNumberHash, _ := hex.DecodeString(tc.RandomNumberHash)
		msg := NewRefundHTLTMsg(tc.From, randomNumberHash)

		err := msg.ValidateBasic()
		if tc.Pass {
			require.Nil(t, err, "test: %v", i)
		} else {
			require.NotNil(t, err, "test: %v", i)
			require.Equal(t, err.Code(), tc.ErrorCode)
		}
	}
}
