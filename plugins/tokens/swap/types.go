package swap

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type SwapStatus int8
type HexData []byte

const (
	NULL      SwapStatus = 0x00
	Open      SwapStatus = 0x01
	Completed SwapStatus = 0x02
	Expired   SwapStatus = 0x03
)

func (hexData HexData) String() string {
	str := hex.EncodeToString(hexData)
	if len(str) == 0 {
		return ""
	}
	return "0x" + str
}

func (hexData HexData) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexData.String())
}

func (hexData *HexData) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if len(s) == 0 {
		*hexData = nil
		return nil
	}
	if !strings.HasPrefix(s, "0x") {
		return fmt.Errorf("hex string must prefix with 0x")
	}
	bytesArray, err := hex.DecodeString(s[2:])
	if err != nil {
		return err
	}
	*hexData = bytesArray
	return nil
}

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

	ExpectedIncome      string  `json:"expected_income"`
	RecipientOtherChain HexData `json:"recipient_other_chain"`

	RandomNumberHash HexData `json:"random_number_hash"` // 32-length byte array, sha256(random_number, timestamp)
	RandomNumber     HexData `json:"random_number"`      // random_number is a 32-length random byte array
	Timestamp        int64   `json:"timestamp"`

	CrossChain bool `json:"cross_chain"`

	ExpireHeight int64      `json:"expire_height"`
	Index        int64      `json:"index"`
	ClosedTime   int64      `json:"closed_time"`
	Status       SwapStatus `json:"status"`
}
