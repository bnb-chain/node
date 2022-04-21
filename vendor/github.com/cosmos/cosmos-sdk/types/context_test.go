package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

type MockLogger struct {
	logs *[]string
}

func NewMockLogger() MockLogger {
	logs := make([]string, 0)
	return MockLogger{
		&logs,
	}
}

func (l MockLogger) Debug(msg string, kvs ...interface{}) {
	*l.logs = append(*l.logs, msg)
}

func (l MockLogger) Info(msg string, kvs ...interface{}) {
	*l.logs = append(*l.logs, msg)
}

func (l MockLogger) Error(msg string, kvs ...interface{}) {
	*l.logs = append(*l.logs, msg)
}

func (l MockLogger) With(kvs ...interface{}) log.Logger {
	panic("not implemented")
}

func defaultContext(key types.StoreKey) types.Context {
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(key, types.StoreTypeIAVL, db)
	cms.LoadLatestVersion()
	ctx := types.NewContext(cms, abci.Header{}, types.RunTxModeDeliver, log.NewNopLogger())
	return ctx
}

func TestCacheContext(t *testing.T) {
	key := types.NewKVStoreKey(t.Name())
	k1 := []byte("hello")
	v1 := []byte("world")
	k2 := []byte("key")
	v2 := []byte("value")

	ctx := defaultContext(key)
	ctx = ctx.WithAccountCache(&types.DummyAccountCache{})

	store := ctx.KVStore(key)
	store.Set(k1, v1)
	require.Equal(t, v1, store.Get(k1))
	require.Nil(t, store.Get(k2))

	cctx, write := ctx.CacheContext()
	cstore := cctx.KVStore(key)
	require.Equal(t, v1, cstore.Get(k1))
	require.Nil(t, cstore.Get(k2))

	cstore.Set(k2, v2)
	require.Equal(t, v2, cstore.Get(k2))
	require.Nil(t, store.Get(k2))

	write()

	require.Equal(t, v2, store.Get(k2))
}

func TestLogContext(t *testing.T) {
	key := types.NewKVStoreKey(t.Name())
	ctx := defaultContext(key)
	logger := NewMockLogger()
	ctx = ctx.WithLogger(logger)
	ctx.Logger().Debug("debug")
	ctx.Logger().Info("info")
	ctx.Logger().Error("error")
	require.Equal(t, *logger.logs, []string{"debug", "info", "error"})
}

// Testing saving/loading sdk type values to/from the context
func TestContextWithCustom(t *testing.T) {
	var ctx types.Context
	require.True(t, ctx.IsZero())

	header := abci.Header{}
	height := int64(1)
	chainid := "chainid"
	logger := NewMockLogger()
	voteinfos := []abci.VoteInfo{{}}

	ctx = types.NewContext(nil, header, types.RunTxModeCheck, logger)
	require.Equal(t, header, ctx.BlockHeader())

	ctx = ctx.
		WithBlockHeight(height).
		WithChainID(chainid).
		WithVoteInfos(voteinfos)
	require.Equal(t, height, ctx.BlockHeight())
	require.Equal(t, chainid, ctx.ChainID())
	require.Equal(t, true, ctx.IsCheckTx())
	require.Equal(t, logger, ctx.Logger())
	require.Equal(t, voteinfos, ctx.VoteInfos())
}

func BenchmarkContext(b *testing.B) {
	ctx := types.NewContext(nil, abci.Header{}, types.RunTxModeDeliver, log.NewNopLogger())
	height := int64(1)
	chainid := "chainid"
	voteinfos := []abci.VoteInfo{{}}
	for n := 0; n < b.N; n++ {
		ctx = ctx.WithBlockHeight(height).
			WithChainID(chainid).
			WithVoteInfos(voteinfos)
	}
	for n := 0; n < b.N; n++ {
		_ = ctx.BlockHeight()
		_ = ctx.ChainID()
		_ = ctx.VoteInfos()
	}
}

func BenchmarkContextWithTx(b *testing.B) {
	ctx := types.NewContext(nil, abci.Header{}, types.RunTxModeDeliver, log.NewNopLogger())
	var tx types.Tx

	priv := ed25519.GenPrivKey()
	addr := types.AccAddress(priv.PubKey().Address())
	msgs := []types.Msg{types.NewTestMsg(addr)}
	sigs := []auth.StdSignature{}
	tx = auth.NewStdTx(msgs, sigs, "", 0, nil)

	height := int64(1)
	chainid := "chainid"
	voteinfos := []abci.VoteInfo{{}}
	for n := 0; n < b.N; n++ {
		ctx = ctx.WithBlockHeight(height).
			WithChainID(chainid).
			WithVoteInfos(voteinfos).
			WithTx(tx)
	}
	for n := 0; n < b.N; n++ {
		_ = ctx.BlockHeight()
		_ = ctx.ChainID()
		_ = ctx.VoteInfos()
		_ = ctx.Tx()
	}
}
