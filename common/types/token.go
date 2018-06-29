package types

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/BiJie/BinanceChain/common/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/tmlibs/common"
)

func makePrivKey(secret common.HexBytes) crypto.PrivKey {
	privKey := crypto.GenPrivKeyEd25519FromSecret(secret)
	return privKey
}

func ValidateSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("token symbol cannot be empty")
	}

	if !utils.IsAlphaNum(symbol) {
		return errors.New("token symbol should be alphanumeric")
	}

	return nil
}

func GenerateTokenAddress(token Token, sequence int64) (sdk.Address, error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(sequence))
	secret := append(token.Owner, b...)
	priv := makePrivKey(secret)
	return priv.PubKey().Address(), nil
}

// we should decide the range of the two variables.
// the length of Name and Symbol also should be limited
type Token struct {
	Name    string      `json:"Name"`
	Symbol  string      `json:"Symbol"`
	Supply  int64       `json:"Supply"`
	Decimal int8        `json:"Decimal"`
	Owner   sdk.Address `json:"From"`
	Address sdk.Address `json:"Address"`
}

func NewToken(name, symbol string, supply int64, decimal int8, owner sdk.Address) Token {
	return Token{
		Name:    name,
		Symbol:  symbol,
		Supply:  supply,
		Decimal: decimal,
		Owner:   owner,
	}
}

func (token *Token) IsOwner(addr sdk.Address) bool     { return bytes.Equal(token.Owner, addr) }
func (token *Token) IsTokenAddr(addr sdk.Address) bool { return bytes.Equal(token.Address, addr) }
func (token *Token) SetAddress(addr sdk.Address)       { token.Address = addr }
func (token Token) String() string {
	return fmt.Sprintf("{Name: %v, Symbol: %v, Supply: %v, Decimal: %v, Address: %X}",
		token.Name, token.Symbol, token.Supply, token.Decimal, token.Address)
}

func (token *Token) Validate() error {
	ValidateSymbol(token.Symbol)

	// TODO: add non-negative check once the type fixed
	return nil
}
