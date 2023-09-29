package airdrop

import (
	"encoding/hex"
	"encoding/json"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	Route   = "airdrop"
	MsgType = "airdrop_approval"
)

var _ sdk.Msg = AirdropApproval{}

func NewAirdropApprovalMsg(tokenIndex uint64, tokenSymbol string, amount uint64, recipient string) AirdropApproval {
	return AirdropApproval{
		TokenIndex:  tokenIndex,
		TokenSymbol: tokenSymbol,
		Amount:      amount,
		Recipient:   recipient,
	}
}

func newAirDropApprovalSignData(tokenIndex uint64, tokenSymbol string, amount uint64, recipient string) airDropApprovalSignData {
	var tokenSymbolBytes [32]byte
	copy(tokenSymbolBytes[:], []byte(tokenSymbol))

	return airDropApprovalSignData{
		TokenIndex:  hex.EncodeToString(big.NewInt(int64(tokenIndex)).FillBytes(make([]byte, 32))),
		TokenSymbol: hex.EncodeToString(tokenSymbolBytes[:]),
		Amount:      hex.EncodeToString(big.NewInt(int64(amount)).FillBytes(make([]byte, 32))),
		Recipient:   recipient,
	}
}

type airDropApprovalSignData struct {
	TokenIndex  string `json:"token_index"`  // hex string(32 bytes)
	TokenSymbol string `json:"token_symbol"` // hex string(32 bytes)
	Amount      string `json:"amount"`       // hex string(32 bytes)
	Recipient   string `json:"recipient"`    // eth address(20 bytes)
}

type AirdropApproval struct {
	TokenIndex  uint64 `json:"token_index"`
	TokenSymbol string `json:"token_symbol"`
	Amount      uint64 `json:"amount"`
	Recipient   string `json:"recipient"` // eth address
}

// GetInvolvedAddresses implements types.Msg.
func (msg AirdropApproval) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

// GetSignBytes implements types.Msg.
func (msg AirdropApproval) GetSignBytes() []byte {
	b, err := json.Marshal(newAirDropApprovalSignData(msg.TokenIndex, msg.TokenSymbol, msg.Amount, msg.Recipient))
	if err != nil {
		panic(err)
	}
	return b
}

// GetSigners implements types.Msg.
func (m AirdropApproval) GetSigners() []sdk.AccAddress {
	// This is not a real on-chain transaction
	// We can get signer from the public key.
	return []sdk.AccAddress{}
}

// Route implements types.Msg.
func (AirdropApproval) Route() string {
	return Route
}

// Type implements types.Msg.
func (AirdropApproval) Type() string {
	return MsgType
}

// ValidateBasic implements types.Msg.
func (msg AirdropApproval) ValidateBasic() sdk.Error {
	if msg.TokenSymbol == "" {
		return sdk.ErrUnknownRequest("Invalid token symbol")
	}

	if msg.Amount == 0 {
		return sdk.ErrUnknownRequest("Invalid amount, should be greater than 0")
	}

	if !common.IsHexAddress(msg.Recipient) {
		return sdk.ErrInvalidAddress("Invalid recipient address")
	}

	return nil
}
