package types

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/utils"
)

const (
	MiniTokenSymbolMaxLen          = 8
	MiniTokenSymbolMinLen          = 3
	MiniTokenSymbolSuffixLen       = 4 // probably enough. if it collides (unlikely) the issuer can just use another tx.
	MiniTokenSymbolTxHashSuffixLen = 3 // probably enough. if it collides (unlikely) the issuer can just use another tx.
	MiniTokenSymbolMSuffix         = "M"

	MiniTokenMinExecutionAmount int64 = 1e8       // 1 with 8 decimal digits
	MiniTokenSupplyUpperBound   int64 = 1000000e8 // 1m with 8 decimal digits
	TinyTokenSupplyUpperBound   int64 = 10000e8  // 10k with 8 decimal digits
	MaxTokenURILength                 = 2048

	TinyRangeType SupplyRangeType = 1
	MiniRangeType SupplyRangeType = 2
)

type SupplyRangeType int8

func (t SupplyRangeType) UpperBound() int64 {
	switch t {
	case TinyRangeType:
		return TinyTokenSupplyUpperBound
	case MiniRangeType:
		return MiniTokenSupplyUpperBound
	default:
		return -1
	}
}

func (t SupplyRangeType) String() string {
	switch t {
	case TinyRangeType:
		return "Tiny"
	case MiniRangeType:
		return "Mini"
	default:
		return "Unknown"
	}
}

var SupplyRange = struct {
	TINY SupplyRangeType
	MINI SupplyRangeType
}{TinyRangeType, MiniRangeType}

type MiniToken struct {
	Name        string          `json:"name"`
	Symbol      string          `json:"symbol"`
	OrigSymbol  string          `json:"original_symbol"`
	TotalSupply utils.Fixed8    `json:"total_supply"`
	Owner       sdk.AccAddress  `json:"owner"`
	Mintable    bool            `json:"mintable"`
	TokenType   SupplyRangeType `json:"token_type"`
	TokenURI    string          `json:"token_uri"` //TODO set max length
}

var _ IToken = &MiniToken{}

func NewMiniToken(name, origSymbol, symbol string, supplyRangeType SupplyRangeType, totalSupply int64, owner sdk.AccAddress, mintable bool, tokenURI string) *MiniToken {
	return &MiniToken{
		Name:        name,
		Symbol:      symbol,
		OrigSymbol:  origSymbol,
		TotalSupply: utils.Fixed8(totalSupply),
		Owner:       owner,
		Mintable:    mintable,
		TokenType:   supplyRangeType,
		TokenURI:    tokenURI,
	}
}

func (token MiniToken) GetName() string {
	return token.Name
}

func (token MiniToken) GetSymbol() string {
	return token.Symbol
}

func (token MiniToken) GetOrigSymbol() string {
	return token.OrigSymbol
}

func (token MiniToken) GetTotalSupply() utils.Fixed8 {
	return token.TotalSupply
}

func (token *MiniToken) SetTotalSupply(totalSupply utils.Fixed8) {
	token.TotalSupply = totalSupply
}

func (token MiniToken) GetOwner() sdk.AccAddress {
	return token.Owner
}

func (token MiniToken) IsMintable() bool {
	return token.Mintable
}

func (token *MiniToken) IsOwner(addr sdk.AccAddress) bool {
	return bytes.Equal(token.Owner, addr)
}

func (token MiniToken) String() string {
	return fmt.Sprintf("{Name: %v, Symbol: %v, TokenType: %v, TotalSupply: %v, Owner: %X, Mintable: %v, TokenURI: %v}",
		token.Name, token.Symbol, token.TokenType, token.TotalSupply, token.Owner, token.Mintable, token.TokenURI)
}

//check if it's mini token by last letter without validation
func IsMiniTokenSymbol(symbol string) bool {
	if symbol == NativeTokenSymbol ||
		symbol == NativeTokenSymbolDotBSuffixed {
		return false
	}
	parts, err := splitSuffixedTokenSymbol(symbol)
	if err != nil {
		return false
	}
	suffixPart := parts[1]

	return len(suffixPart) == MiniTokenSymbolSuffixLen && strings.HasSuffix(suffixPart, MiniTokenSymbolMSuffix)
}

//Validate and check if it's mini token
func IsValidMiniTokenSymbol(symbol string) bool {
	return ValidateMiniTokenSymbol(symbol) == nil
}

func ValidateIssueMiniSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("token symbol cannot be empty")
	}

	if symbol == NativeTokenSymbol ||
		symbol == NativeTokenSymbolDotBSuffixed {
		return errors.New("symbol cannot be the same as native token")
	}

	// check len without suffix
	if symbolLen := len(symbol); symbolLen > MiniTokenSymbolMaxLen || symbolLen < MiniTokenSymbolMinLen {
		return errors.New("length of token symbol is limited to 3~8")
	}

	if !utils.IsAlphaNum(symbol) {
		return errors.New("token symbol should be alphanumeric")
	}

	return nil
}

func ValidateMiniTokenSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("suffixed token symbol cannot be empty")
	}

	if symbol == NativeTokenSymbol ||
		symbol == NativeTokenSymbolDotBSuffixed {
		return errors.New("symbol cannot be the same as native token")
	}

	parts, err := splitSuffixedTokenSymbol(symbol)
	if err != nil {
		return err
	}

	symbolPart := parts[0]
	// check len without suffix
	if len(symbolPart) < MiniTokenSymbolMinLen {
		return fmt.Errorf("mini-token symbol part is too short, got %d chars", len(symbolPart))
	}
	if len(symbolPart) > MiniTokenSymbolMaxLen {
		return fmt.Errorf("mini-token symbol part is too long, got %d chars", len(symbolPart))
	}

	if !utils.IsAlphaNum(symbolPart) {
		return errors.New("mini-token symbol part should be alphanumeric")
	}

	suffixPart := parts[1]
	if len(suffixPart) != MiniTokenSymbolSuffixLen {
		return fmt.Errorf("mini-token symbol suffix must be %d chars in length, got %d", MiniTokenSymbolSuffixLen, len(suffixPart))
	}

	if suffixPart[len(suffixPart)-1:] != MiniTokenSymbolMSuffix {
		return fmt.Errorf("mini-token symbol suffix must end with M")
	}

	// prohibit non-hexadecimal chars in the suffix part
	isHex, err := regexp.MatchString(fmt.Sprintf("[0-9A-F]{%d}M", MiniTokenSymbolTxHashSuffixLen), suffixPart)
	if err != nil {
		return err
	}
	if !isHex {
		return fmt.Errorf("mini-token symbol tx hash suffix must be hex with a length of %d", MiniTokenSymbolTxHashSuffixLen)
	}

	return nil
}
