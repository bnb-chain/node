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

var _ sdk.Msg = HTLTMsg{}

type HTLTMsg struct {
	From                sdk.AccAddress `json:"from"`
	To                  sdk.AccAddress `json:"to"`
	RecipientOtherChain HexData        `json:"recipient_other_chain"`
	RandomNumberHash    HexData        `json:"random_number_hash"`
	Timestamp           int64          `json:"timestamp"`
	OutAmount           sdk.Coins      `json:"out_amount"`
	ExpectedIncome      string         `json:"expected_income"`
	HeightSpan          int64          `json:"height_span"`
	CrossChain          bool           `json:"cross_chain"`
}

func NewHTLTMsg(from, to sdk.AccAddress, recipientOtherChain []byte, randomNumberHash []byte, timestamp int64,
	outAmount sdk.Coins, expectedIncome string, heightSpan int64, crossChain bool) HTLTMsg {
	return HTLTMsg{
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

func (msg HTLTMsg) Route() string { return AtomicSwapRoute }
func (msg HTLTMsg) Type() string  { return HTLT }
func (msg HTLTMsg) String() string {
	return fmt.Sprintf("HTLT{%v#%v#%v#%v#%v#%v#%v#%v#%v}", msg.From, msg.To, msg.RecipientOtherChain, msg.RandomNumberHash,
		msg.Timestamp, msg.OutAmount, msg.ExpectedIncome, msg.HeightSpan, msg.CrossChain)
}
func (msg HTLTMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg HTLTMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg HTLTMsg) ValidateBasic() sdk.Error {
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

func (msg HTLTMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

var _ sdk.Msg = DepositHTLTMsg{}

type DepositHTLTMsg struct {
	From             sdk.AccAddress `json:"from"`
	To               sdk.AccAddress `json:"to"`
	OutAmount        sdk.Coins      `json:"out_amount"`
	RandomNumberHash HexData        `json:"random_number_hash"`
}

func NewDepositHTLTMsg(from, to sdk.AccAddress, outAmount sdk.Coins, randomNumberHash []byte) DepositHTLTMsg {
	return DepositHTLTMsg{
		From:             from,
		To:               to,
		OutAmount:        outAmount,
		RandomNumberHash: randomNumberHash,
	}
}

func (msg DepositHTLTMsg) Route() string { return AtomicSwapRoute }
func (msg DepositHTLTMsg) Type() string  { return DepositHTLT }
func (msg DepositHTLTMsg) String() string {
	return fmt.Sprintf("depositHTLT{%v#%v#%v#%v}", msg.From, msg.To, msg.OutAmount, msg.RandomNumberHash)
}
func (msg DepositHTLTMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg DepositHTLTMsg) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.From}
}

func (msg DepositHTLTMsg) ValidateBasic() sdk.Error {
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

func (msg DepositHTLTMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

var _ sdk.Msg = ClaimHTLTMsg{}

type ClaimHTLTMsg struct {
	From             sdk.AccAddress `json:"from"`
	RandomNumberHash HexData        `json:"random_number_hash"`
	RandomNumber     HexData        `json:"random_number"`
}

func NewClaimHTLTMsg(from sdk.AccAddress, randomNumberHash, randomNumber []byte) ClaimHTLTMsg {
	return ClaimHTLTMsg{
		From:             from,
		RandomNumberHash: randomNumberHash,
		RandomNumber:     randomNumber,
	}
}

func (msg ClaimHTLTMsg) Route() string { return AtomicSwapRoute }
func (msg ClaimHTLTMsg) Type() string  { return ClaimHTLT }
func (msg ClaimHTLTMsg) String() string {
	return fmt.Sprintf("claimHTLT{%v#%v#%v}", msg.From, msg.RandomNumberHash, msg.RandomNumber)
}
func (msg ClaimHTLTMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg ClaimHTLTMsg) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.From}
}

func (msg ClaimHTLTMsg) ValidateBasic() sdk.Error {
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

func (msg ClaimHTLTMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

var _ sdk.Msg = RefundHTLTMsg{}

type RefundHTLTMsg struct {
	From             sdk.AccAddress `json:"from"`
	RandomNumberHash HexData        `json:"random_number_hash"`
}

func NewRefundHTLTMsg(from sdk.AccAddress, randomNumberHash []byte) RefundHTLTMsg {
	return RefundHTLTMsg{
		From:             from,
		RandomNumberHash: randomNumberHash,
	}
}

func (msg RefundHTLTMsg) Route() string { return AtomicSwapRoute }
func (msg RefundHTLTMsg) Type() string  { return RefundHTLT }
func (msg RefundHTLTMsg) String() string {
	return fmt.Sprintf("refundHTLT{%v#%v}", msg.From, msg.RandomNumberHash)
}
func (msg RefundHTLTMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return append(msg.GetSigners(), AtomicSwapCoinsAccAddr)
}
func (msg RefundHTLTMsg) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.From}
}

func (msg RefundHTLTMsg) ValidateBasic() sdk.Error {
	if len(msg.From) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Expected address length is %d, actual length is %d", sdk.AddrLen, len(msg.From)))
	}
	if len(msg.RandomNumberHash) != RandomNumberHashLength {
		return ErrInvalidRandomNumberHash(fmt.Sprintf("The length of random number hash should be %d", RandomNumberHashLength))
	}
	return nil
}

func (msg RefundHTLTMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}
