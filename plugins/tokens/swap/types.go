package swap

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cmm "github.com/tendermint/tendermint/libs/common"
)

type SwapStatus int8

const (
	NULL      SwapStatus = 0x00
	Open      SwapStatus = 0x01
	Completed SwapStatus = 0x02
	Expired   SwapStatus = 0x03
)

func NewSwapStatusFromString(str string) SwapStatus {
	switch str {
	case "Open", "open":
		return Open
	case "Completed", "completed":
		return Completed
	case "Expired", "expired":
		return Expired
	default:
		return NULL
	}
}

func (status SwapStatus) String() string {
	switch status {
	case Open:
		return "Open"
	case Completed:
		return "Completed"
	case Expired:
		return "Expired"
	default:
		return "NULL"
	}
}

func (status SwapStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(status.String())
}

func (status *SwapStatus) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*status = NewSwapStatusFromString(s)
	return nil
}

type AtomicSwap struct {
	From      sdk.AccAddress `json:"from"`
	To        sdk.AccAddress `json:"to"`
	OutAmount sdk.Coins      `json:"out_amount"`
	InAmount  sdk.Coins      `json:"in_amount"`

	ExpectedIncome      string `json:"expected_income"`
	RecipientOtherChain string `json:"recipient_other_chain"`

	RandomNumberHash cmm.HexBytes `json:"random_number_hash"` // 32-length byte array, sha256(random_number, timestamp)
	RandomNumber     cmm.HexBytes `json:"random_number"`      // random_number is a 32-length random byte array
	Timestamp        int64        `json:"timestamp"`

	CrossChain bool `json:"cross_chain"`

	ExpireHeight int64      `json:"expire_height"`
	Index        int64      `json:"index"`
	ClosedTime   int64      `json:"closed_time"`
	Status       SwapStatus `json:"status"`
}
