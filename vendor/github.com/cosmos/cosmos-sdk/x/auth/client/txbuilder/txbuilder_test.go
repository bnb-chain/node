package context

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	priv = ed25519.GenPrivKey()
	addr = sdk.AccAddress(priv.PubKey().Address())
)

func TestTxBuilderBuild(t *testing.T) {
	type fields struct {
		Codec         *codec.Codec
		AccountNumber int64
		Sequence      int64
		ChainID       string
		Memo          string
		Source        int64
	}
	defaultMsg := []sdk.Msg{sdk.NewTestMsg(addr)}
	tests := []struct {
		fields  fields
		msgs    []sdk.Msg
		want    StdSignMsg
		wantErr bool
	}{
		{
			fields{
				Codec:         codec.New(),
				AccountNumber: 1,
				Sequence:      1,
				ChainID:       "test-chain",
				Memo:          "hello",
				Source:        0,
			},
			defaultMsg,
			StdSignMsg{
				ChainID:       "test-chain",
				AccountNumber: 1,
				Sequence:      1,
				Memo:          "hello",
				Msgs:          defaultMsg,
				Source:        0,
			},
			false,
		},
	}
	for i, tc := range tests {
		bldr := TxBuilder{
			Codec:         tc.fields.Codec,
			AccountNumber: tc.fields.AccountNumber,
			Sequence:      tc.fields.Sequence,
			ChainID:       tc.fields.ChainID,
			Memo:          tc.fields.Memo,
			Source:        tc.fields.Source,
		}
		got, err := bldr.Build(tc.msgs)
		require.Equal(t, tc.wantErr, (err != nil), "TxBuilder.Build() error = %v, wantErr %v, tc %d", err, tc.wantErr, i)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("TxBuilder.Build() = %v, want %v", got, tc.want)
		}
	}
}
