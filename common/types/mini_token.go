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

	MiniTokenMinTotalSupply   int64 = 100000000      // 1 with 8 decimal digits
	MiniTokenSupplyUpperBound int64 = 10000000000000 // 100k with 8 decimal digits
	TinyTokenSupplyUpperBound int64 = 1000000000000
	MaxTokenURILength               = 2048

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
}{TinyRangeType, MiniRangeType }

type MiniToken struct {
	Name        string          `json:"name"`
	Symbol      string          `json:"symbol"`
	OrigSymbol  string          `json:"original_symbol"`
	TokenType   SupplyRangeType `json:"token_type"`
	TotalSupply utils.Fixed8    `json:"total_supply"`
	Owner       sdk.AccAddress  `json:"owner"`
	Mintable    bool            `json:"mintable"`
	TokenURI    string          `json:"token_uri"` //TODO set max length
}

func NewMiniToken(name, symbol string, supplyRangeType int8, totalSupply int64, owner sdk.AccAddress, mintable bool, tokenURI string) (*MiniToken, error) {
	// double check that the symbol is suffixed
	if err := ValidateMapperMiniTokenSymbol(symbol); err != nil {
		return nil, err
	}
	parts, err := splitSuffixedMiniTokenSymbol(symbol)
	if err != nil {
		return nil, err
	}
	return &MiniToken{
		Name:        name,
		Symbol:      symbol,
		OrigSymbol:  parts[0],
		TokenType:   SupplyRangeType(supplyRangeType),
		TotalSupply: utils.Fixed8(totalSupply),
		Owner:       owner,
		Mintable:    mintable,
		TokenURI:    tokenURI,
	}, nil
}

func IsMiniTokenSymbol(symbol string) bool {
	if err := ValidateMapperMiniTokenSymbol(symbol); err != nil {
		return false
	}
	return true
}

func (token *MiniToken) IsOwner(addr sdk.AccAddress) bool { return bytes.Equal(token.Owner, addr) }
func (token MiniToken) String() string {
	return fmt.Sprintf("{Name: %v, Symbol: %v, TokenType: %v, TotalSupply: %v, Owner: %X, Mintable: %v, TokenURI: %v}",
		token.Name, token.Symbol, token.TokenType, token.TotalSupply, token.Owner, token.Mintable, token.TokenURI)
}

// Token Validation

func ValidateMiniToken(token MiniToken) error {
	if err := ValidateMapperMiniTokenSymbol(token.Symbol); err != nil {
		return err
	}
	if err := ValidateIssueMsgMiniTokenSymbol(token.OrigSymbol); err != nil {
		return err
	}
	return nil
}

func ValidateIssueMsgMiniTokenSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("token symbol cannot be empty")
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

func ValidateMapperMiniTokenSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("suffixed token symbol cannot be empty")
	}

	parts, err := splitSuffixedMiniTokenSymbol(symbol)
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

func splitSuffixedMiniTokenSymbol(suffixed string) ([]string, error) {

	split := strings.SplitN(suffixed, "-", 2)

	if len(split) != 2 {
		return nil, errors.New("suffixed mini-token symbol must contain a hyphen ('-')")
	}

	if strings.Contains(split[1], "-") {
		return nil, errors.New("suffixed mini-token symbol must contain just one hyphen ('-')")
	}

	return split, nil
}
