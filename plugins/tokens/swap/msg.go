package swap

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	AtomicSwapRoute = "atomicSwap"
	HTLT            = "HTLT"
	DepositHTLT     = "depositHTLT"
	ClaimHTLT       = "claimHTLT"
	RefundHTLT      = "refundHTLT"

	RandomNumberHashLength       = 32
	RandomNumberLength           = 32
	MaxRecipientOtherChainLength = 32
	MaxExpectedIncomeLength      = 64
	MinimumHeightSpan            = 360
	MaximumHeightSpan            = 518400
)

var _ sdk.Msg = HashTimerLockTransferMsg{}

type HashTimerLockTransferMsg struct {
	From                sdk.AccAddress `json:"from"`
	To                  sdk.AccAddress `json:"to"`
	RecipientOtherChain HexData        `json:"recipient_other_chain"`
	RandomNumberHash    HexData        `json:"random_number_hash"`
	Timestamp           int64          `json:"timestamp"`
	OutAmount           sdk.Coin       `json:"out_amount"`
	ExpectedIncome      string         `json:"expected_income"`
	HeightSpan          int64          `json:"height_span"`
	CrossChain          bool           `json:"cross_chain"`
}

func NewHashTimerLockTransferMsg(from, to sdk.AccAddress, recipientOtherChain []byte, randomNumberHash []byte, timestamp int64,
	outAmount sdk.Coin, expectedIncome string, heightSpan int64, crossChain bool) HashTimerLockTransferMsg {
	return HashTimerLockTransferMsg{
		From:                from,
		To:                  to,
		RecipientOtherChain: recipientOtherChain,
		RandomNumberHash:    randomNumberHash,
		Timestamp:           timestamp,
		OutAmount:           outAmount,
		ExpectedIncome:      expectedIncome,
		HeightSpan:          heightSpan,
		CrossChain:          crossChain,
	}
}

func (msg HashTimerLockTransferMsg) Route() string { return AtomicSwapRoute }
func (msg HashTimerLockTransferMsg) Type() string  { return HTLT }
func (msg HashTimerLockTransferMsg) String() string {
	return fmt.Sprintf("HTLT{%v#%v#%v#%v#%v#%v#%v#%v#%v}", msg.From, msg.To, msg.RecipientOtherChain, msg.RandomNumberHash,
		msg.Timestamp, msg.OutAmount, msg.ExpectedIncome, msg.HeightSpan, msg.CrossChain)
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
	if !msg.CrossChain && len(msg.RecipientOtherChain) != 0 {
		return ErrInvalidRecipientAddrOtherChain("Must leave recipient address on other chain to empty for single chain swap")
	}
	if msg.CrossChain && len(msg.RecipientOtherChain) == 0 {
		return ErrInvalidRecipientAddrOtherChain("Missing recipient address on other chain for cross chain swap")
	}
	if len(msg.RecipientOtherChain) > MaxRecipientOtherChainLength {
		return ErrInvalidRecipientAddrOtherChain(fmt.Sprintf("The length of recipient address on other chain should be less than %d", MaxRecipientOtherChainLength))
	}
	if len(msg.ExpectedIncome) > MaxExpectedIncomeLength {
		return ErrExpectedIncomeTooLong(fmt.Sprintf("The length of expected income should be less than %d", MaxExpectedIncomeLength))
	}
	if len(msg.RandomNumberHash) != RandomNumberHashLength {
		return ErrInvalidRandomNumberHash(fmt.Sprintf("The length of random number hash should be %d", RandomNumberHashLength))
	}
	if !msg.OutAmount.IsPositive() {
		return ErrInvalidSwapOutAmount("The swapped out coin must be positive")
	}
	if msg.HeightSpan < MinimumHeightSpan || msg.HeightSpan > MaximumHeightSpan {
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

var _ sdk.Msg = DepositHashTimerLockMsg{}

type DepositHashTimerLockMsg struct {
	From             sdk.AccAddress `json:"from"`
	To               sdk.AccAddress `json:"to"`
	OutAmount        sdk.Coin       `json:"out_amount"`
	RandomNumberHash HexData        `json:"random_number_hash"`
}

func NewDepositHashTimerLockMsg(from, to sdk.AccAddress, outAmount sdk.Coin, randomNumberHash []byte) DepositHashTimerLockMsg {
	return DepositHashTimerLockMsg{
		From:             from,
		To:               to,
		OutAmount:        outAmount,
		RandomNumberHash: randomNumberHash,
	}
}

func (msg DepositHashTimerLockMsg) Route() string { return AtomicSwapRoute }
func (msg DepositHashTimerLockMsg) Type() string  { return DepositHTLT }
func (msg DepositHashTimerLockMsg) String() string {
	return fmt.Sprintf("depositHTLT{%v#%v#%v#%v}", msg.From, msg.To, msg.OutAmount, msg.RandomNumberHash)
}
func (msg DepositHashTimerLockMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg DepositHashTimerLockMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg DepositHashTimerLockMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.From)))
	}
	if len(msg.To) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.To)))
	}
	if len(msg.RandomNumberHash) != RandomNumberHashLength {
		return ErrInvalidRandomNumberHash(fmt.Sprintf("The length of random number hash should be %d", RandomNumberHashLength))
	}
	if !msg.OutAmount.IsPositive() {
		return ErrInvalidSwapOutAmount("The swapped out coin must be positive")
	}
	return nil
}

func (msg DepositHashTimerLockMsg) GetSignBytes() []byte {
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
