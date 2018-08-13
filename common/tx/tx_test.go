package tx_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"

	txns "github.com/BiJie/BinanceChain/common/tx"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestStdTx(t *testing.T) {
	priv := ed25519.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())
	msg := txns.NewTestMsg(addr)
	fee := newStdFee()
	sigs := []txns.StdSignature{}

	tx := txns.NewStdTx(msg, fee, sigs, "")
	require.Equal(t, msg, tx.GetMsg())
	require.Equal(t, sigs, tx.GetSignatures())

	feePayer := txns.FeePayer(tx)
	require.Equal(t, addr, feePayer)
}

func TestStdSignBytes(t *testing.T) {
	priv := ed25519.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())
	msg := txns.NewTestMsg(addr)
	fee := newStdFee()
	signMsg := txns.StdSignMsg{
		"1234",
		3,
		6,
		fee,
		msg,
		"memo",
	}
	require.Equal(t, fmt.Sprintf("{\"account_number\":\"3\",\"chain_id\":\"1234\",\"fee\":{\"amount\":[{\"amount\":\"150\",\"denom\":\"atom\"}],\"gas\":\"5000\"},\"memo\":\"memo\",\"msg\":[\"%s\"],\"sequence\":\"6\"}", addr), string(signMsg.Bytes()))
}
