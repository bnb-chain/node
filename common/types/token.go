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
	TokenSymbolMaxLen          = 8
	TokenSymbolMinLen          = 3
	TokenSymbolTxHashSuffixLen = 3 // probably enough. if it collides (unlikely) the issuer can just use another tx.
	TokenSymbolDotBSuffix      = ".B"

	TokenDecimals       int8  = 8
	TokenMaxTotalSupply int64 = 9000000000000000000 // 90 billions with 8 decimal digits

	NativeTokenSymbol             = "BNB" // number of zeros = TokenSymbolTxHashSuffixLen
	NativeTokenSymbolDotBSuffixed = "BNB" + TokenSymbolDotBSuffix
	NativeTokenTotalSupply        = 2e16
)

type Token struct {
	Name        string         `json:"name"`
	Symbol      string         `json:"symbol"`
	OrigSymbol  string         `json:"original_symbol"`
	TotalSupply utils.Fixed8   `json:"total_supply"`
	Owner       sdk.AccAddress `json:"owner"`
	Mintable    bool           `json:"mintable"`
}

func NewToken(name, symbol string, totalSupply int64, owner sdk.AccAddress, mintable bool) (*Token, error) {
	// double check that the symbol is suffixed
	if err := ValidateMapperTokenSymbol(symbol); err != nil {
		return nil, err
	}
	parts, err := splitSuffixedTokenSymbol(symbol)
	if err != nil {
		return nil, err
	}
	return &Token{
		Name:        name,
		Symbol:      symbol,
		OrigSymbol:  parts[0],
		TotalSupply: utils.Fixed8(totalSupply),
		Owner:       owner,
		Mintable:    mintable,
	}, nil
}

func (token *Token) IsOwner(addr sdk.AccAddress) bool { return bytes.Equal(token.Owner, addr) }
func (token Token) String() string {
	return fmt.Sprintf("{Name: %v, Symbol: %v, TotalSupply: %v, Owner: %X, Mintable: %v}",
		token.Name, token.Symbol, token.TotalSupply, token.Owner, token.Mintable)
}

// Token Validation

func ValidateToken(token Token) error {
	if err := ValidateMapperTokenSymbol(token.Symbol); err != nil {
		return err
	}
	if err := ValidateIssueMsgTokenSymbol(token.OrigSymbol); err != nil {
		return err
	}
	return nil
}

func ValidateIssueMsgTokenSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("token symbol cannot be empty")
	}

	if strings.HasSuffix(symbol, TokenSymbolDotBSuffix) {
		symbol = strings.TrimSuffix(symbol, TokenSymbolDotBSuffix)
	}

	// check len without .B suffix
	if symbolLen := len(symbol); symbolLen > TokenSymbolMaxLen || symbolLen < TokenSymbolMinLen {
		return errors.New("length of token symbol is limited to 3~8")
	}

	if !utils.IsAlphaNum(symbol) {
		return errors.New("token symbol should be alphanumeric")
	}

	return nil
}

func ValidateMapperTokenCoins(coins sdk.Coins) error {
	for _, coin := range coins {
		err := ValidateMapperTokenSymbol(coin.Denom)
		if err != nil {
			return err
		}
	}
	return nil
}

func ValidateMapperTokenSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("suffixed token symbol cannot be empty")
	}

	// suffix exception for native token (less drama in existing tests)
	if symbol == NativeTokenSymbol ||
		symbol == NativeTokenSymbolDotBSuffixed {
		return nil
	}

	parts, err := splitSuffixedTokenSymbol(symbol)
	if err != nil {
		return err
	}

	symbolPart := parts[0]

	// since the native token was given a suffix exception above, do not allow it to have a suffix
	if symbolPart == NativeTokenSymbol ||
		symbolPart == NativeTokenSymbolDotBSuffixed {
		return errors.New("native token symbol should not be suffixed with tx hash")
	}

	if strings.HasSuffix(symbolPart, TokenSymbolDotBSuffix) {
		symbolPart = strings.TrimSuffix(symbolPart, TokenSymbolDotBSuffix)
	}

	// check len without .B suffix
	if len(symbolPart) < TokenSymbolMinLen {
		return fmt.Errorf("token symbol part is too short, got %d chars", len(symbolPart))
	}
	if len(symbolPart) > TokenSymbolMaxLen {
		return fmt.Errorf("token symbol part is too long, got %d chars", len(symbolPart))
	}

	if !utils.IsAlphaNum(symbolPart) {
		return errors.New("token symbol part should be alphanumeric")
	}

	txHashPart := parts[1]

	if len(txHashPart) != TokenSymbolTxHashSuffixLen {
		return fmt.Errorf("token symbol tx hash suffix must be %d chars in length, got %d", TokenSymbolTxHashSuffixLen, len(txHashPart))
	}

	// prohibit non-hexadecimal chars in the suffix part
	isHex, err := regexp.MatchString(fmt.Sprintf("[0-9A-F]{%d}", TokenSymbolTxHashSuffixLen), txHashPart)
	if err != nil {
		return err
	}
	if !isHex {
		return fmt.Errorf("token symbol tx hash suffix must be hex with a length of %d", TokenSymbolTxHashSuffixLen)
	}

	return nil
}

func splitSuffixedTokenSymbol(suffixed string) ([]string, error) {
	// as above, the native token symbol is given an exception - it is not required to be suffixed
	if suffixed == NativeTokenSymbol ||
		suffixed == NativeTokenSymbolDotBSuffixed {
		return []string{suffixed, ""}, nil
	}

	split := strings.SplitN(suffixed, "-", 2)

	if len(split) != 2 {
		return nil, errors.New("suffixed token symbol must contain a hyphen ('-')")
	}

	if strings.Contains(split[1], "-") {
		return nil, errors.New("suffixed token symbol must contain just one hyphen ('-')")
	}

	return split, nil
}
