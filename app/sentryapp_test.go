package app

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/BiJie/BinanceChain/common/log"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randBytes(n int) []byte {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

func applyBlock(t *testing.T, app *SentryApplication, checkTxs [][]byte, recheckTxs [][]byte, recheckRes []bool, cacheSize int, msg string) {
	assert := assert.New(t)
	app.BeginBlock(abci.RequestBeginBlock{})
	for _, tx := range checkTxs {
		resp := app.CheckTx(tx)
		assert.Equal(abci.CodeTypeOK, resp.Code, msg)
	}
	app.EndBlock(abci.RequestEndBlock{})
	app.Commit()
	for i, tx := range recheckTxs {
		resp := app.ReCheckTx(tx)
		assert.Equal(recheckRes[i], resp.Code == abci.CodeTypeOK, msg)
	}
	assert.Equal(cacheSize, len(app.cache.pool), msg)
}

func Test_SentryApplication(t *testing.T) {
	assert := assert.New(t)
	app := NewSentryApplication(log.NewAsyncFileLogger(os.DevNull, 1024), nil, nil)
	sentryApp := app.(*SentryApplication)
	checkTxs := make([][]byte, 1000)
	for i := 0; i < 1000; i++ {
		checkTxs[i] = randBytes(10)
	}
	recheckTx := make([][]byte, 10000)
	for i := 0; i < 10000; i++ {
		recheckTx[i] = randBytes(10)
	}
	genRecheckRes := func(n, start, end int) []bool {
		assert.True(n > 0)
		assert.True(start >= 0)
		assert.True(end <= n)
		res := make([]bool, n)
		for i := start; i < end; i++ {
			res[i] = true
		}
		return res
	}
	testCases := []struct {
		checkTxs   [][]byte
		recheckTxs [][]byte
		recheckRes []bool
		cacheSize  int
		msg        string
	}{
		{checkTxs: checkTxs, recheckTxs: recheckTx[:1000], recheckRes: genRecheckRes(1000, 0, 1000), cacheSize: 1000, msg: "step 1"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[1000:2000], recheckRes: genRecheckRes(1000, 0, 1000), cacheSize: 2000, msg: "step 2"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[2000:3000], recheckRes: genRecheckRes(1000, 0, 1000), cacheSize: 2000, msg: "step 3"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[3000:4000], recheckRes: genRecheckRes(1000, 0, 1000), cacheSize: 2000, msg: "step 4"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[4000:5000], recheckRes: genRecheckRes(1000, 0, 1000), cacheSize: 2000, msg: "step 5"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[5000:6000], recheckRes: genRecheckRes(1000, 0, 1000), cacheSize: 2000, msg: "step 6"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[5500:6000], recheckRes: genRecheckRes(1000, 0, 1000), cacheSize: 1000, msg: "step 7"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[5500:6000], recheckRes: genRecheckRes(500, 0, 500), cacheSize: 500, msg: "step 8"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[5500:6000], recheckRes: genRecheckRes(500, 0, 500), cacheSize: 500, msg: "step 9"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[5500:6000], recheckRes: genRecheckRes(500, 0, 500), cacheSize: 500, msg: "step 10"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[5500:6000], recheckRes: genRecheckRes(500, 0, 500), cacheSize: 500, msg: "step 11"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[6000:6500], recheckRes: genRecheckRes(500, 0, 500), cacheSize: 1000, msg: "step 12"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[6500:7000], recheckRes: genRecheckRes(500, 0, 500), cacheSize: 1000, msg: "step 13"},
		{checkTxs: checkTxs, recheckTxs: recheckTx[7000:7500], recheckRes: genRecheckRes(500, 0, 500), cacheSize: 1000, msg: "step 14"},
		{checkTxs: checkTxs, recheckTxs: nil, recheckRes: nil, cacheSize: 500, msg: "step 15"},
		{checkTxs: checkTxs, recheckTxs: nil, recheckRes: nil, cacheSize: 0, msg: "step 16"},
	}
	for _, c := range testCases {
		applyBlock(t, sentryApp, c.checkTxs, c.recheckTxs, c.recheckRes, c.cacheSize, c.msg)
	}
	for i := 0; i < DefaultMaxSurvive-1; i++ {
		applyBlock(t, sentryApp, checkTxs, recheckTx[8000:9000], genRecheckRes(1000, 0, 1000), 1000, fmt.Sprintf("survive %d", i+1))
	}
	applyBlock(t, sentryApp, checkTxs, recheckTx[8000:9000], genRecheckRes(1000, 0, 0), 0, fmt.Sprintf("survive %d", DefaultMaxSurvive))

}
