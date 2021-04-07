package types

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/common/utils"
)

const (
	TokenSymbolMaxLen          = 8
	TokenSymbolMinLen          = 3
	TokenSymbolNewMinLen       = 2
	TokenSymbolTxHashSuffixLen = 3 // probably enough. if it collides (unlikely) the issuer can just use another tx.
	TokenSymbolDotBSuffix      = ".B"

	TokenDecimals       int8  = 8
	TokenMaxTotalSupply int64 = 9000000000000000000 // 90 billions with 8 decimal digits

	NativeTokenSymbol             = "BNB" // number of zeros = TokenSymbolTxHashSuffixLen
	NativeTokenSymbolDotBSuffixed = "BNB" + TokenSymbolDotBSuffix
	NativeTokenTotalSupply        = 2e16
)

type IToken interface {
	GetName() string
	GetSymbol() string
	GetOrigSymbol() string
	GetTotalSupply() utils.Fixed8
	SetTotalSupply(totalSupply utils.Fixed8)
	SetContractAddress(addr string)
	GetContractAddress() string
	SetContractDecimals(decimal int8)
	GetContractDecimals() int8

	SetOwner(addr sdk.AccAddress)
	GetOwner() sdk.AccAddress
	IsMintable() bool
	IsOwner(addr sdk.AccAddress) bool
	String() string
}

var _ IToken = &Token{}

type Token struct {
	Name             string         `json:"name"`
	Symbol           string         `json:"symbol"`
	OrigSymbol       string         `json:"original_symbol"`
	TotalSupply      utils.Fixed8   `json:"total_supply"`
	Owner            sdk.AccAddress `json:"owner"`
	Mintable         bool           `json:"mintable"`
	ContractAddress  string         `json:"contract_address,omitempty"`
	ContractDecimals int8           `json:"contract_decimals,omitempty"`
}

func (token Token) GetName() string {
	return token.Name
}

func (token *Token) SetContractDecimals(decimal int8) {
	token.ContractDecimals = decimal
}

func (token Token) GetContractDecimals() int8 {
	return token.ContractDecimals
}

func (token *Token) SetContractAddress(addr string) {
	token.ContractAddress = addr
}

func (token *Token) GetContractAddress() string {
	return token.ContractAddress
}

func (token Token) GetSymbol() string {
	return token.Symbol
}

func (token Token) GetOrigSymbol() string {
	return token.OrigSymbol
}

func (token Token) GetTotalSupply() utils.Fixed8 {
	return token.TotalSupply
}

func (token *Token) SetTotalSupply(totalSupply utils.Fixed8) {
	token.TotalSupply = totalSupply
}

func (token *Token) SetOwner(addr sdk.AccAddress) {
	cp := make(sdk.AccAddress, len(addr.Bytes()))
	copy(cp, addr)
	token.Owner = cp
}

func (token Token) GetOwner() sdk.AccAddress {
	return token.Owner
}

func (token Token) IsMintable() bool {
	return token.Mintable
}

func NewToken(name, symbol string, totalSupply int64, owner sdk.AccAddress, mintable bool) (*Token, error) {
	// double check that the symbol is suffixed
	if err := ValidateTokenSymbol(symbol); err != nil {
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

// This function is used by both client and server side, and the client needs to use TokenSymbolNewMinLen for the validation.
// If the UpgradeMgr.GetHeight == 0, that indicates the function is invoked by client side, and we should use TokenSymbolNewMinLen
func ValidateIssueSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("token symbol cannot be empty")
	}

	if strings.HasSuffix(symbol, TokenSymbolDotBSuffix) {
		symbol = strings.TrimSuffix(symbol, TokenSymbolDotBSuffix)
	}

	// check len without .B suffix
	symbolLen := len(symbol)
	if sdk.UpgradeMgr.GetHeight() == 0 || sdk.IsUpgrade(upgrade.BEP87) {
		if symbolLen > TokenSymbolMaxLen || symbolLen < TokenSymbolNewMinLen {
			return errors.New("length of token symbol is limited to 2~8")
		}
	} else if symbolLen > TokenSymbolMaxLen || symbolLen < TokenSymbolMinLen {
		return errors.New("length of token symbol is limited to 3~8")
	}

	if !utils.IsAlphaNum(symbol) {
		return errors.New("token symbol should be alphanumeric")
	}

	return nil
}

func ValidateTokenSymbols(coins sdk.Coins) error {
	for _, coin := range coins {
		err := ValidateTokenSymbol(coin.Denom)
		if err != nil {
			return err
		}
	}
	return nil
}

func ValidateTokenSymbol(symbol string) error {
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
	// This function is used by both client and server side, and the client needs to use TokenSymbolNewMinLen for the validation.
	// If the UpgradeMgr.GetHeight == 0, that indicates the function is invoked by client side, and we should use TokenSymbolNewMinLen
	if sdk.UpgradeMgr.GetHeight() == 0 || sdk.IsUpgrade(upgrade.BEP87) {
		if len(symbolPart) < TokenSymbolNewMinLen {
			return fmt.Errorf("token symbol part is too short, got %d chars", len(symbolPart))
		}
	} else if len(symbolPart) < TokenSymbolMinLen {
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
