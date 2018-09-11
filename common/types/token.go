package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/common"

	"github.com/BiJie/BinanceChain/common/utils"
)

const (
	Decimals       int8  = 8
	MaxTotalSupply int64 = 9000000000000000000 // 90 billions with 8 decimal digits

	DotBSuffix  = ".B"
	NativeToken = "BNB"
)

type Token struct {
	Name        string         `json:"name"`
	Symbol      string         `json:"symbol"`
	TotalSupply utils.Fixed8   `json:"total_supply"`
	Owner       sdk.AccAddress `json:"owner"`
}

func NewToken(name, symbol string, totalSupply int64, owner sdk.AccAddress) Token {
	return Token{
		Name:        name,
		Symbol:      symbol,
		TotalSupply: utils.Fixed8(totalSupply),
		Owner:       owner,
	}
}

func (token *Token) IsOwner(addr sdk.AccAddress) bool { return bytes.Equal(token.Owner, addr) }
func (token Token) String() string {
	return fmt.Sprintf("{Name: %v, Symbol: %v, TotalSupply: %v, Owner: %X}",
		token.Name, token.Symbol, token.TotalSupply, token.Owner)
}

func ValidateSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("token symbol cannot be empty")
	}

	if len(symbol) > 8 {
		return errors.New("token symbol is too long")
	}

	if strings.HasSuffix(symbol, DotBSuffix) {
		symbol = strings.TrimSuffix(symbol, DotBSuffix)
	}

	if !utils.IsAlphaNum(symbol) {
		return errors.New("token symbol should be alphanumeric")
	}

	return nil
}

func GenerateTokenAddress(token Token, sequence int64) (sdk.AccAddress, error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(sequence))
	secret := append(token.Owner, b...)
	priv := makePrivKey(common.HexBytes(secret))
	return sdk.AccAddress(priv.PubKey().Address()), nil
}

func makePrivKey(secret common.HexBytes) crypto.PrivKey {
	privKey := ed25519.GenPrivKeyFromSecret(secret)
	return privKey
}
