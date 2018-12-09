package store_test

import (
	"testing"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
)

func TestValidatePairSymbol(t *testing.T) {
	type args struct {
		symbol string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// happy
		{
			name:    "Valid pair symbol 1",
			args:    args{"1YZ-000000_" + types.NativeTokenSymbol},
			wantErr: false,
		},
		{
			name:    "Valid pair symbol 2",
			args:    args{"XYZ.B-000000_" + types.NativeTokenSymbol},
			wantErr: false,
		},
		{
			name:    "Valid pair symbol 3",
			args:    args{"XYZ.B-000000_" + types.NativeTokenSymbolDotBSuffixed},
			wantErr: false,
		},
		{
			name:    "Valid pair symbol 4",
			args:    args{"XYZ.B-000000_BNX-000000"},
			wantErr: false,
		},
		{
			name:    "Valid pair symbol 5",
			args:    args{"XYZ.B-000000_BNX.B-000000"},
			wantErr: false,
		},
		{
			name:    "Valid pair symbol 5",
			args:    args{"12345678.B-000000_12345678.B-000000"},
			wantErr: false,
		},
		// bad
		{
			name:    "Invalid pair symbol - too long",
			args:    args{"12345678.B-000000_12345678.B-000000A"},
			wantErr: true,
		},
		{
			name:    "Invalid pair symbol 1",
			args:    args{"XYZ-000000_ABC-000000_" + types.NativeTokenSymbol},
			wantErr: true,
		},
		{
			name:    "Invalid pair symbol 2",
			args:    args{"XYZ.B-000000_BNB.Z-000000"},
			wantErr: true,
		},
		{
			name:    "Invalid pair symbol 3",
			args:    args{"XYZ.B-000000_" + types.NativeTokenSymbol + ".Z"},
			wantErr: true,
		},
		{
			name:    "Invalid pair symbol 4",
			args:    args{"XYZ-000000_BN$-000000"},
			wantErr: true,
		},
		{
			name:    "Invalid pair symbol 5",
			args:    args{"XYZ-000000_ABC-0000001"},
			wantErr: true,
		},
		{
			name:    "Invalid pair symbol 6",
			args:    args{"XYZ-00000_ABC-000000"},
			wantErr: true,
		},
		{
			name:    "Invalid pair symbol 7",
			args:    args{"XYZ-X00000_ABC-000000"},
			wantErr: true,
		},
		{
			name:    "Invalid pair symbol 8",
			args:    args{"XYZ-000000_ABC456789-000000"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.ValidatePairSymbol(tt.args.symbol); (err != nil) != tt.wantErr {
				t.Errorf("ValidatePairSymbol() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
