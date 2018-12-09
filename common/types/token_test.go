package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/types"
)

var issueMsgSymbolTestCases = []struct {
	symbol  string
	correct bool
}{
	// happy
	{types.NativeTokenSymbol, true}, // BNB
	{types.NativeTokenSymbolDotBSuffixed, true}, // BNB.B
	{"XYZ", true},
	{"XYZ45678", true},
	{"XYZ45678.B", true}, // still ok - .B suffix extends max len by suffix len
	// sad
	{"XYZ456789.B", false}, // too long
	{"XYZ45678.C", false},
	{"XYZ456789", false},
	{types.NativeTokenSymbol + ".C", false},
	{"#@#$", false},
}

var tokenMapperSymbolTestCases = []struct {
	symbol  string
	correct bool
}{
	// happy
	{types.NativeTokenSymbol, true}, // BNB
	{types.NativeTokenSymbolDotBSuffixed, true}, // BNB.B
	{"XYZ45678-000000", true},
	{"XYZ-000000", true},
	{"1YZ-000000", true},
	{"XYZ.B-000000", true},
	{"XYZ.B-000000", true},
	// sad
	{types.NativeTokenSymbol+"-000000", false}, // no tx hash suffix for native token
	{types.NativeTokenSymbolDotBSuffixed+"-000000", false},
	{"XYZ-00000", false},
	{"XYZ-0000000", false},
	{"XYZ-X00000", false},
	{"XYZ-$00000", false},
	{"XYZ-000000-111111", false},
	{"XYZ.C-000000", false},
	{"XYZ.B-X00000", false},
	{"XYZ.B-$00000", false},
	{"XYZ.B-00000", false},
	{"XYZ.B-0000000", false},
	{"XYZ456789-0000000", false},
	{"XYZ456789.B-0000000", false},
}

func TestNewToken(t *testing.T) {
	for _, tt := range tokenMapperSymbolTestCases {
		t.Run(tt.symbol, func(t *testing.T) {
			_, err := types.NewToken(tt.symbol, tt.symbol, 100000, sdk.AccAddress{})
			if (err == nil) != tt.correct {
				t.Errorf("NewToken() error = %v, correct %v", err, tt.correct)
				return
			}
		})
	}
	// extra test. an orig symbol that is valid in TestValidateIssueMsgTokenSymbol but not here
	_, err := types.NewToken("XYZ", "XYZ", 100000, sdk.AccAddress{})
	if err == nil {
		t.Errorf("NewToken() error = %v, expected XYZ to be invalid", err)
	}
}

func TestValidateIssueMsgTokenSymbol(t *testing.T) {
	for _, tt := range issueMsgSymbolTestCases {
		t.Run(tt.symbol, func(t *testing.T) {
			if err := types.ValidateIssueMsgTokenSymbol(tt.symbol); (err == nil) != tt.correct {
				t.Errorf("ValidateIssueMsgTokenSymbol() error = %v, correct %v", err, tt.correct)
			}
		})
	}
	// extra test. an issued symbol that is valid in NewToken and ValidateMapperTokenSymbol but not here
	err := types.ValidateIssueMsgTokenSymbol("XYZ-000000")
	if err == nil {
		t.Errorf("ValidateIssueMsgTokenSymbol() error = %v, expected XYZ-000000 to be invalid", err)
	}
}

func TestValidateMapperTokenSymbol(t *testing.T) {
	for _, tt := range tokenMapperSymbolTestCases {
		t.Run(tt.symbol, func(t *testing.T) {
			if err := types.ValidateMapperTokenSymbol(tt.symbol); (err == nil) != tt.correct {
				t.Errorf("ValidateMapperTokenSymbol() error = %v, correct %v", err, tt.correct)
			}
		})
	}
	// extra test. an issued symbol that is valid in ValidateIssueMsgTokenSymbol but not here
	err := types.ValidateMapperTokenSymbol("XYZ")
	if err == nil {
		t.Errorf("ValidateIssueMsgTokenSymbol() error = %v, expected XYZ to be invalid", err)
	}
}
