package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

var issueMsgSymbolTestCases = []struct {
	symbol  string
	correct bool
}{
	// happy
	{types.NativeTokenSymbol, true},             // BNB
	{types.NativeTokenSymbolDotBSuffixed, true}, // BNB.B
	{"XYZ", true},
	{"XYZ45678", true},
	{"XYZ45678.B", true}, // still ok - .B suffix extends max len by suffix len
	// sad
	{"XY", false},
	{"XY.B", false},
	{"#@#$", false},
	{"#@#$.B", false},
	{"XYZ.B.B", false},
	{"XYZ456789.B", false}, // too long
	{"XYZ45678.C", false},
	{"XYZ456789", false},
	{types.NativeTokenSymbol + ".C", false},
}

var tokenMapperSymbolTestCases = []struct {
	symbol  string
	correct bool
}{
	// happy
	{types.NativeTokenSymbol, true},             // BNB
	{types.NativeTokenSymbolDotBSuffixed, true}, // BNB.B
	{"XYZ45678-000", true},
	{"XYZ-000", true},
	{"1YZ-000", true},
	{"XYZ.B-000", true},
	// sad
	{types.NativeTokenSymbol + "-000", false}, // no tx hash suffix for native token
	{types.NativeTokenSymbolDotBSuffixed + "-000", false},
	{"XY-000", false},
	{"XY.B-000", false},
	{"#@#$-000", false},
	{"#@#$.B-000", false},
	{"XYZ.B.B-000", false},
	{"XYZ-00", false},   // 2 != 3
	{"XYZ-0000", false}, // 4 != 3
	{"XYZ-X00", false},
	{"XYZ-$00", false},
	{"XYZ-000-111", false},
	{"XYZ.C-000", false},
	{"XYZ.B-X00", false},
	{"XYZ.B-$00", false},
	{"XYZ.B-00", false},
	{"XYZ.B-0000", false},
	{"XYZ456789-000", false},
	{"XYZ456789.B-000", false},
}

func TestNewToken(t *testing.T) {
	for _, tt := range tokenMapperSymbolTestCases {
		t.Run(tt.symbol, func(t *testing.T) {
			_, err := types.NewToken(tt.symbol, tt.symbol, 100000, sdk.AccAddress{}, false)
			if (err == nil) != tt.correct {
				t.Errorf("NewToken() error = %v, correct %v", err, tt.correct)
				return
			}
		})
	}
	// extra test. an orig symbol that is valid in TestValidateIssueMsgTokenSymbol but not here
	if _, err := types.NewToken("XYZ", "XYZ", 100000, sdk.AccAddress{}, false); err == nil {
		t.Errorf("NewToken() error = %v, expected XYZ to be invalid", err)
	}
}

func TestValidateIssueSymbol(t *testing.T) {
	for _, tt := range issueMsgSymbolTestCases {
		t.Run(tt.symbol, func(t *testing.T) {
			if err := types.ValidateIssueSymbol(tt.symbol); (err == nil) != tt.correct {
				t.Errorf("ValidateIssueSymbol() error = %v, correct %v", err, tt.correct)
			}
		})
	}
	// extra test. an issued symbol that is valid in NewToken and ValidateTokenSymbol but not here
	if err := types.ValidateIssueSymbol("XYZ-000"); err == nil {
		t.Errorf("ValidateIssueSymbol() error = %v, expected XYZ-000 to be invalid", err)
	}
}

func TestValidateTokenSymbol(t *testing.T) {
	for _, tt := range tokenMapperSymbolTestCases {
		t.Run(tt.symbol, func(t *testing.T) {
			if err := types.ValidateTokenSymbol(tt.symbol); (err == nil) != tt.correct {
				t.Errorf("ValidateTokenSymbol() error = %v, correct %v", err, tt.correct)
			}
		})
	}
	// extra test. an issued symbol that is valid in ValidateIssueSymbol but not here
	if err := types.ValidateTokenSymbol("XYZ"); err == nil {
		t.Errorf("ValidateIssueSymbol() error = %v, expected XYZ to be invalid", err)
	}
}
