package swap

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	AtomicSwapRoute       = "atomicSwap"
	HashTimerLockTransfer = "hashTimerLockTransfer"
	ClaimHashTimeLock     = "claimHashTimeLock"
	RefundLockedAsset     = "refundLockedAsset"
)

var _ sdk.Msg = HashTimerLockTransferMsg{}

type HashTimerLockTransferMsg struct {
	From             sdk.AccAddress `json:"from"`
	To               sdk.AccAddress `json:"to"`
	ToOnOtherChain   HexData        `json:"to_on_other_chain"`
	RandomNumberHash HexData        `json:"random_number_hash"`
	Timestamp        uint64         `json:"timestamp"`
	OutAmount        sdk.Coin       `json:"out_amount"`
	InAmount         uint64         `json:"in_amount"`
	TimeSpan         uint64         `json:"time_span"`
}

func NewHashTimerLockTransferMsg(from, to sdk.AccAddress, toOnOtherChain []byte, randomNumberHash []byte, timestamp uint64,
	outAmount sdk.Coin, inAmount uint64, timespan uint64) HashTimerLockTransferMsg {
	return HashTimerLockTransferMsg{
		From:             from,
		To:               to,
		ToOnOtherChain:   toOnOtherChain,
		RandomNumberHash: randomNumberHash,
		Timestamp:        timestamp,
		OutAmount:        outAmount,
		InAmount:         inAmount,
		TimeSpan:         timespan,
	}
}

func (msg HashTimerLockTransferMsg) Route() string { return AtomicSwapRoute }
func (msg HashTimerLockTransferMsg) Type() string  { return HashTimerLockTransfer }
func (msg HashTimerLockTransferMsg) String() string {
	return fmt.Sprintf("hashTimerLockTransfer{%v#%v#%v#%v#%v#%v#%v#%v}", msg.From, msg.To, msg.ToOnOtherChain, msg.RandomNumberHash,
		msg.Timestamp, msg.OutAmount, msg.InAmount, msg.TimeSpan)
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
	if msg.TimeSpan < 360 || msg.TimeSpan > 518400 {
		return ErrInvalidTimeSpan("The timespan should no less than 360 and no greater than 518400")
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
func (msg ClaimHashTimerLockMsg) Type() string  { return ClaimHashTimeLock }
func (msg ClaimHashTimerLockMsg) String() string {
	return fmt.Sprintf("claimHashTimeLock{%v#%v#%v}", msg.From, msg.RandomNumberHash, msg.RandomNumber)
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

var _ sdk.Msg = RefundLockedAssetMsg{}

type RefundLockedAssetMsg struct {
	From             sdk.AccAddress `json:"from"`
	RandomNumberHash HexData        `json:"random_number_hash"`
}

func NewRefundLockedAssetMsg(from sdk.AccAddress, randomNumberHash []byte) RefundLockedAssetMsg {
	return RefundLockedAssetMsg{
		From:             from,
		RandomNumberHash: randomNumberHash,
	}
}

func (msg RefundLockedAssetMsg) Route() string { return AtomicSwapRoute }
func (msg RefundLockedAssetMsg) Type() string  { return RefundLockedAsset }
func (msg RefundLockedAssetMsg) String() string {
	return fmt.Sprintf("refundLockedAsset{%v#%v}", msg.From, msg.RandomNumberHash)
}
func (msg RefundLockedAssetMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg RefundLockedAssetMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg RefundLockedAssetMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.From)))
	}
	if len(msg.RandomNumberHash) != RandomNumberHashLength {
		return ErrInvalidRandomNumberHash(fmt.Sprintf("The length of random number hash should be %d", RandomNumberHashLength))
	}
	return nil
}

func (msg RefundLockedAssetMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}
