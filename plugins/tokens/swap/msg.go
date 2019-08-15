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

var _ sdk.Msg = HashTimerLockedTransferMsg{}

type HashTimerLockedTransferMsg struct {
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

func NewHashTimerLockedTransferMsg(from, to sdk.AccAddress, recipientOtherChain []byte, randomNumberHash []byte, timestamp int64,
	outAmount sdk.Coin, expectedIncome string, heightSpan int64, crossChain bool) HashTimerLockedTransferMsg {
	return HashTimerLockedTransferMsg{
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

func (msg HashTimerLockedTransferMsg) Route() string { return AtomicSwapRoute }
func (msg HashTimerLockedTransferMsg) Type() string  { return HTLT }
func (msg HashTimerLockedTransferMsg) String() string {
	return fmt.Sprintf("HTLT{%v#%v#%v#%v#%v#%v#%v#%v#%v}", msg.From, msg.To, msg.RecipientOtherChain, msg.RandomNumberHash,
		msg.Timestamp, msg.OutAmount, msg.ExpectedIncome, msg.HeightSpan, msg.CrossChain)
}
func (msg HashTimerLockedTransferMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg HashTimerLockedTransferMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg HashTimerLockedTransferMsg) ValidateBasic() sdk.Error {
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

func (msg HashTimerLockedTransferMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

var _ sdk.Msg = DepositHashTimerLockedTransferMsg{}

type DepositHashTimerLockedTransferMsg struct {
	From             sdk.AccAddress `json:"from"`
	To               sdk.AccAddress `json:"to"`
	OutAmount        sdk.Coin       `json:"out_amount"`
	RandomNumberHash HexData        `json:"random_number_hash"`
}

func NewDepositHashTimerLockedTransferMsg(from, to sdk.AccAddress, outAmount sdk.Coin, randomNumberHash []byte) DepositHashTimerLockedTransferMsg {
	return DepositHashTimerLockedTransferMsg{
		From:             from,
		To:               to,
		OutAmount:        outAmount,
		RandomNumberHash: randomNumberHash,
	}
}

func (msg DepositHashTimerLockedTransferMsg) Route() string { return AtomicSwapRoute }
func (msg DepositHashTimerLockedTransferMsg) Type() string  { return DepositHTLT }
func (msg DepositHashTimerLockedTransferMsg) String() string {
	return fmt.Sprintf("depositHTLT{%v#%v#%v#%v}", msg.From, msg.To, msg.OutAmount, msg.RandomNumberHash)
}
func (msg DepositHashTimerLockedTransferMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg DepositHashTimerLockedTransferMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg DepositHashTimerLockedTransferMsg) ValidateBasic() sdk.Error {
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

func (msg DepositHashTimerLockedTransferMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

var _ sdk.Msg = ClaimHashTimerLockedTransferMsg{}

type ClaimHashTimerLockedTransferMsg struct {
	From             sdk.AccAddress `json:"from"`
	RandomNumberHash HexData        `json:"random_number_hash"`
	RandomNumber     HexData        `json:"random_number"`
}

func NewClaimHashTimerLockedTransferMsg(from sdk.AccAddress, randomNumberHash, randomNumber []byte) ClaimHashTimerLockedTransferMsg {
	return ClaimHashTimerLockedTransferMsg{
		From:             from,
		RandomNumberHash: randomNumberHash,
		RandomNumber:     randomNumber,
	}
}

func (msg ClaimHashTimerLockedTransferMsg) Route() string { return AtomicSwapRoute }
func (msg ClaimHashTimerLockedTransferMsg) Type() string  { return ClaimHTLT }
func (msg ClaimHashTimerLockedTransferMsg) String() string {
	return fmt.Sprintf("claimHTLT{%v#%v#%v}", msg.From, msg.RandomNumberHash, msg.RandomNumber)
}
func (msg ClaimHashTimerLockedTransferMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg ClaimHashTimerLockedTransferMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg ClaimHashTimerLockedTransferMsg) ValidateBasic() sdk.Error {
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

func (msg ClaimHashTimerLockedTransferMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

var _ sdk.Msg = RefundHashTimerLockedTransferMsg{}

type RefundHashTimerLockedTransferMsg struct {
	From             sdk.AccAddress `json:"from"`
	RandomNumberHash HexData        `json:"random_number_hash"`
}

func NewRefundRefundHashTimerLockedTransferMsg(from sdk.AccAddress, randomNumberHash []byte) RefundHashTimerLockedTransferMsg {
	return RefundHashTimerLockedTransferMsg{
		From:             from,
		RandomNumberHash: randomNumberHash,
	}
}

func (msg RefundHashTimerLockedTransferMsg) Route() string { return AtomicSwapRoute }
func (msg RefundHashTimerLockedTransferMsg) Type() string  { return RefundHTLT }
func (msg RefundHashTimerLockedTransferMsg) String() string {
	return fmt.Sprintf("refundHTLT{%v#%v}", msg.From, msg.RandomNumberHash)
}
func (msg RefundHashTimerLockedTransferMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg RefundHashTimerLockedTransferMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg RefundHashTimerLockedTransferMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.From)))
	}
	if len(msg.RandomNumberHash) != RandomNumberHashLength {
		return ErrInvalidRandomNumberHash(fmt.Sprintf("The length of random number hash should be %d", RandomNumberHashLength))
	}
	return nil
}

func (msg RefundHashTimerLockedTransferMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}
