package swap

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
)

const (
	AtomicSwapRoute = "atomicSwap"
	HTLT            = "HTLT"
	DepositHTLT     = "depositHTLT"
	ClaimHTLT       = "claimHTLT"
	RefundHTLT      = "refundHTLT"

	RandomNumberHashLength  = 32
	RandomNumberLength      = 32
	SwapIDLength            = 32
	MaxOtherChainAddrLength = 64
	MaxExpectedIncomeLength = 64
	MinimumHeightSpan       = 360
	MaximumHeightSpan       = 518400
)

var _ sdk.Msg = HTLTMsg{}

type HTLTMsg struct {
	From                sdk.AccAddress `json:"from"`
	To                  sdk.AccAddress `json:"to"`
	RecipientOtherChain string         `json:"recipient_other_chain"`
	SenderOtherChain    string         `json:"sender_other_chain"`
	RandomNumberHash    SwapBytes      `json:"random_number_hash"`
	Timestamp           int64          `json:"timestamp"`
	Amount              sdk.Coins      `json:"amount"`
	ExpectedIncome      string         `json:"expected_income"`
	HeightSpan          int64          `json:"height_span"`
	CrossChain          bool           `json:"cross_chain"`
}

func NewHTLTMsg(from, to sdk.AccAddress, recipientOtherChain, senderOtherChain string, randomNumberHash SwapBytes, timestamp int64,
	amount sdk.Coins, expectedIncome string, heightSpan int64, crossChain bool) HTLTMsg {
	return HTLTMsg{
		From:                from,
		To:                  to,
		RecipientOtherChain: recipientOtherChain,
		SenderOtherChain:    senderOtherChain,
		RandomNumberHash:    randomNumberHash,
		Timestamp:           timestamp,
		Amount:              amount,
		ExpectedIncome:      expectedIncome,
		HeightSpan:          heightSpan,
		CrossChain:          crossChain,
	}
}

func (msg HTLTMsg) Route() string { return AtomicSwapRoute }
func (msg HTLTMsg) Type() string  { return HTLT }
func (msg HTLTMsg) String() string {
	return fmt.Sprintf("HTLT{%v#%v#%v#%v#%v#%v#%v#%v#%v#%v}", msg.From, msg.To, msg.RecipientOtherChain, msg.SenderOtherChain, msg.RandomNumberHash,
		msg.Timestamp, msg.Amount, msg.ExpectedIncome, msg.HeightSpan, msg.CrossChain)
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
		return ErrInvalidAddrOtherChain("Must leave recipient address on other chain to empty for single chain swap")
	}
	if !msg.CrossChain && len(msg.SenderOtherChain) != 0 {
		return ErrInvalidAddrOtherChain("Must leave sender address on other chain to empty for single chain swap")
	}
	if msg.CrossChain && len(msg.RecipientOtherChain) == 0 {
		return ErrInvalidAddrOtherChain("Missing recipient address on other chain for cross chain swap")
	}
	if len(msg.RecipientOtherChain) > MaxOtherChainAddrLength {
		return ErrInvalidAddrOtherChain(fmt.Sprintf("The length of recipient address on other chain should be less than %d", MaxOtherChainAddrLength))
	}
	if len(msg.SenderOtherChain) > MaxOtherChainAddrLength {
		return ErrInvalidAddrOtherChain(fmt.Sprintf("The length of sender address on other chain should be less than %d", MaxOtherChainAddrLength))
	}
	if len(msg.ExpectedIncome) > MaxExpectedIncomeLength {
		return ErrInvalidExpectedIncome(fmt.Sprintf("The length of expected income should be less than %d", MaxExpectedIncomeLength))
	}
	if len(msg.RandomNumberHash) != RandomNumberHashLength {
		return ErrInvalidRandomNumberHash(fmt.Sprintf("The length of random number hash should be %d", RandomNumberHashLength))
	}
	if !msg.Amount.IsPositive() {
		return sdk.ErrInvalidCoins("The swapped out coins must be positive")
	}
	if msg.HeightSpan < MinimumHeightSpan || msg.HeightSpan > MaximumHeightSpan {
		return ErrInvalidHeightSpan("The height span should be no less than 360 and no greater than 518400")
	}

	if sdk.IsUpgrade(upgrade.BEP8) {
		symbolError := types.ValidateTokenSymbols(msg.Amount)
		if symbolError != nil {
			return sdk.ErrInvalidCoins(symbolError.Error())
		}
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
	From   sdk.AccAddress `json:"from"`
	Amount sdk.Coins      `json:"amount"`
	SwapID SwapBytes      `json:"swap_id"`
}

func NewDepositHTLTMsg(from sdk.AccAddress, amount sdk.Coins, swapID SwapBytes) DepositHTLTMsg {
	return DepositHTLTMsg{
		From:   from,
		Amount: amount,
		SwapID: swapID,
	}
}

func (msg DepositHTLTMsg) Route() string { return AtomicSwapRoute }
func (msg DepositHTLTMsg) Type() string  { return DepositHTLT }
func (msg DepositHTLTMsg) String() string {
	return fmt.Sprintf("depositHTLT{%v#%v#%v}", msg.From, msg.Amount, msg.SwapID)
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
	if len(msg.SwapID) != SwapIDLength {
		return ErrInvalidSwapID(fmt.Sprintf("The length of swapID should be %d", SwapIDLength))
	}
	if !msg.Amount.IsPositive() {
		return sdk.ErrInvalidCoins("The swapped out coins must be positive")
	}
	if sdk.IsUpgrade(upgrade.BEP8) {
		symbolError := types.ValidateTokenSymbols(msg.Amount)
		if symbolError != nil {
			return sdk.ErrInvalidCoins(symbolError.Error())
		}
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
	From         sdk.AccAddress `json:"from"`
	SwapID       SwapBytes      `json:"swap_id"`
	RandomNumber SwapBytes      `json:"random_number"`
}

func NewClaimHTLTMsg(from sdk.AccAddress, swapID, randomNumber SwapBytes) ClaimHTLTMsg {
	return ClaimHTLTMsg{
		From:         from,
		SwapID:       swapID,
		RandomNumber: randomNumber,
	}
}

func (msg ClaimHTLTMsg) Route() string { return AtomicSwapRoute }
func (msg ClaimHTLTMsg) Type() string  { return ClaimHTLT }
func (msg ClaimHTLTMsg) String() string {
	return fmt.Sprintf("claimHTLT{%v#%v#%v}", msg.From, msg.SwapID, msg.RandomNumber)
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
	if len(msg.SwapID) != SwapIDLength {
		return ErrInvalidSwapID(fmt.Sprintf("The length of swapID should be %d", SwapIDLength))
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
	From   sdk.AccAddress `json:"from"`
	SwapID SwapBytes      `json:"swap_id"`
}

func NewRefundHTLTMsg(from sdk.AccAddress, swapID SwapBytes) RefundHTLTMsg {
	return RefundHTLTMsg{
		From:   from,
		SwapID: swapID,
	}
}

func (msg RefundHTLTMsg) Route() string { return AtomicSwapRoute }
func (msg RefundHTLTMsg) Type() string  { return RefundHTLT }
func (msg RefundHTLTMsg) String() string {
	return fmt.Sprintf("refundHTLT{%v#%v}", msg.From, msg.SwapID)
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
	if len(msg.SwapID) != SwapIDLength {
		return ErrInvalidSwapID(fmt.Sprintf("The length of swapID should be %d", SwapIDLength))
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
