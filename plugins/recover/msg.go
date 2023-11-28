package recover

import (
	"encoding/hex"
	"encoding/json"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	Route   = "recover"
	MsgType = "request_token"
)

var _ sdk.Msg = TokenRecoverRequest{}

func NewTokenRecoverRequestMsg(tokenSymbol string, amount uint64, recipient string) TokenRecoverRequest {
	return TokenRecoverRequest{
		TokenSymbol: tokenSymbol,
		Amount:      amount,
		Recipient:   recipient,
	}
}

func newTokenRecoverRequestSignData(tokenSymbol string, amount uint64, recipient string) tokenRecoverRequestSignData {
	var tokenSymbolBytes [32]byte
	copy(tokenSymbolBytes[:], []byte(tokenSymbol))

	return tokenRecoverRequestSignData{
		TokenSymbol: hex.EncodeToString(tokenSymbolBytes[:]),
		Amount:      hex.EncodeToString(big.NewInt(int64(amount)).FillBytes(make([]byte, 32))),
		Recipient:   recipient,
	}
}

type tokenRecoverRequestSignData struct {
	TokenSymbol string `json:"token_symbol"` // hex string(32 bytes)
	Amount      string `json:"amount"`       // hex string(32 bytes)
	Recipient   string `json:"recipient"`    // eth address(20 bytes)
}

type TokenRecoverRequest struct {
	TokenSymbol string `json:"token_symbol"`
	Amount      uint64 `json:"amount"`
	Recipient   string `json:"recipient"` // eth address
}

// GetInvolvedAddresses implements types.Msg.
func (msg TokenRecoverRequest) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

// GetSignBytes implements types.Msg.
func (msg TokenRecoverRequest) GetSignBytes() []byte {
	b, err := json.Marshal(newTokenRecoverRequestSignData(msg.TokenSymbol, msg.Amount, msg.Recipient))
	if err != nil {
		panic(err)
	}
	return b
}

// GetSigners implements types.Msg.
func (m TokenRecoverRequest) GetSigners() []sdk.AccAddress {
	// This is not a real on-chain transaction
	// We can get signer from the public key.
	return []sdk.AccAddress{}
}

// Route implements types.Msg.
func (TokenRecoverRequest) Route() string {
	return Route
}

// Type implements types.Msg.
func (TokenRecoverRequest) Type() string {
	return MsgType
}

// ValidateBasic implements types.Msg.
func (msg TokenRecoverRequest) ValidateBasic() sdk.Error {
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
