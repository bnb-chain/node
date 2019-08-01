package swap

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	AtomicSwapRoute = "atomicSwap"
	HTLT            = "HTLT"
	ClaimHTLT       = "claimHTLT"
	RefundHTLT      = "refundHTLT"
)

var _ sdk.Msg = HashTimerLockTransferMsg{}

type HashTimerLockTransferMsg struct {
	From             sdk.AccAddress `json:"from"`
	To               sdk.AccAddress `json:"to"`
	ToOnOtherChain   HexData        `json:"to_on_other_chain"`
	RandomNumberHash HexData        `json:"random_number_hash"`
	Timestamp        int64          `json:"timestamp"`
	OutAmount        sdk.Coin       `json:"out_amount"`
	InAmount         int64          `json:"in_amount"`
	HeightSpan       int64          `json:"height_span"`
}

func NewHashTimerLockTransferMsg(from, to sdk.AccAddress, toOnOtherChain []byte, randomNumberHash []byte, timestamp int64,
	outAmount sdk.Coin, inAmount int64, heightSpan int64) HashTimerLockTransferMsg {
	return HashTimerLockTransferMsg{
		From:             from,
		To:               to,
		ToOnOtherChain:   toOnOtherChain,
		RandomNumberHash: randomNumberHash,
		Timestamp:        timestamp,
		OutAmount:        outAmount,
		InAmount:         inAmount,
		HeightSpan:       heightSpan,
	}
}

func (msg HashTimerLockTransferMsg) Route() string { return AtomicSwapRoute }
func (msg HashTimerLockTransferMsg) Type() string  { return HTLT }
func (msg HashTimerLockTransferMsg) String() string {
	return fmt.Sprintf("HTLT{%v#%v#%v#%v#%v#%v#%v#%v}", msg.From, msg.To, msg.ToOnOtherChain, msg.RandomNumberHash,
		msg.Timestamp, msg.OutAmount, msg.InAmount, msg.HeightSpan)
}
func (msg HashTimerLockTransferMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg HashTimerLockTransferMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg HashTimerLockTransferMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.From)))
	}
	if len(msg.To) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.To)))
	}
	if len(msg.ToOnOtherChain) == 0 || len(msg.ToOnOtherChain) > 32 {
		return ErrInvalidOtherChainAddress("The receiver address on other chain shouldn't be nil and its length shouldn't exceed 32")
	}
	if len(msg.RandomNumberHash) != RandomNumberHashLength {
		return ErrInvalidRandomNumberHash(fmt.Sprintf("The length of random number hash should be %d", RandomNumberHashLength))
	}
	if !msg.OutAmount.IsPositive() {
		return ErrInvalidSwapOutAmount("The swapped out coin must be positive")
	}
	if msg.HeightSpan < 360 || msg.HeightSpan > 518400 {
		return ErrInvalidHeightSpan("The height span should be no less than 360 and no greater than 518400")
	}
	return nil
}

func (msg HashTimerLockTransferMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

var _ sdk.Msg = ClaimHashTimerLockMsg{}

type ClaimHashTimerLockMsg struct {
	From             sdk.AccAddress `json:"from"`
	RandomNumberHash HexData        `json:"random_number_hash"`
	RandomNumber     HexData        `json:"random_number"`
}

func NewClaimHashTimerLockMsg(from sdk.AccAddress, randomNumberHash, randomNumber []byte) ClaimHashTimerLockMsg {
	return ClaimHashTimerLockMsg{
		From:             from,
		RandomNumberHash: randomNumberHash,
		RandomNumber:     randomNumber,
	}
}

func (msg ClaimHashTimerLockMsg) Route() string { return AtomicSwapRoute }
func (msg ClaimHashTimerLockMsg) Type() string  { return ClaimHTLT }
func (msg ClaimHashTimerLockMsg) String() string {
	return fmt.Sprintf("claimHTLT{%v#%v#%v}", msg.From, msg.RandomNumberHash, msg.RandomNumber)
}
func (msg ClaimHashTimerLockMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg ClaimHashTimerLockMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg ClaimHashTimerLockMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.From)))
	}
	if len(msg.RandomNumberHash) != RandomNumberHashLength {
		return ErrInvalidRandomNumberHash(fmt.Sprintf("The length of random number hash should be %d", RandomNumberHashLength))
	}
	if len(msg.RandomNumber) != RandomNumberLength {
		return ErrInvalidRandomNumber(fmt.Sprintf("The length of random number should be %d", RandomNumberLength))
	}
	return nil
}

func (msg ClaimHashTimerLockMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

var _ sdk.Msg = RefundHashTimerLockMsg{}

type RefundHashTimerLockMsg struct {
	From             sdk.AccAddress `json:"from"`
	RandomNumberHash HexData        `json:"random_number_hash"`
}

func NewRefundLockedAssetMsg(from sdk.AccAddress, randomNumberHash []byte) RefundHashTimerLockMsg {
	return RefundHashTimerLockMsg{
		From:             from,
		RandomNumberHash: randomNumberHash,
	}
}

func (msg RefundHashTimerLockMsg) Route() string { return AtomicSwapRoute }
func (msg RefundHashTimerLockMsg) Type() string  { return RefundHTLT }
func (msg RefundHashTimerLockMsg) String() string {
	return fmt.Sprintf("refundHTLT{%v#%v}", msg.From, msg.RandomNumberHash)
}
func (msg RefundHashTimerLockMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg RefundHashTimerLockMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg RefundHashTimerLockMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.From)))
	}
	if len(msg.RandomNumberHash) != RandomNumberHashLength {
		return ErrInvalidRandomNumberHash(fmt.Sprintf("The length of random number hash should be %d", RandomNumberHashLength))
	}
	return nil
}

func (msg RefundHashTimerLockMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}
