package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	priv = ed25519.GenPrivKey()
	addr = sdk.AccAddress(priv.PubKey().Address())
)

func TestStdTx(t *testing.T) {
	msgs := []sdk.Msg{sdk.NewTestMsg(addr)}
	sigs := []StdSignature{}

	tx := NewStdTx(msgs, sigs, "", 0, nil)
	require.Equal(t, msgs, tx.GetMsgs())
	require.Equal(t, sigs, tx.GetSignatures())
}

func TestStdSignBytes(t *testing.T) {
	type args struct {
		chainID  string
		accnum   int64
		sequence int64
		msgs     []sdk.Msg
		memo     string
		source   int64
		data     []byte
	}
	tests := []struct {
		args args
		want string
	}{
		{
			args{"1234", 3, 6, []sdk.Msg{sdk.NewTestMsg(addr)}, "memo", 0, nil},
			fmt.Sprintf("{\"account_number\":\"3\",\"chain_id\":\"1234\",\"data\":null,\"memo\":\"memo\",\"msgs\":[[\"%s\"]],\"sequence\":\"6\",\"source\":\"0\"}", addr),
		},
	}
	for i, tc := range tests {
		got := string(StdSignBytes(tc.args.chainID, tc.args.accnum, tc.args.sequence, tc.args.msgs, tc.args.memo, tc.args.source, tc.args.data))
		require.Equal(t, tc.want, got, "Got unexpected result on test case i: %d", i)
	}
}
