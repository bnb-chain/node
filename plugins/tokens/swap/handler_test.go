package swap

import (
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/testutils"
)

func setup() (sdk.Context, sdk.Handler, Keeper, auth.AccountKeeper) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)
	handler := NewHandler(keeper)

	return ctx, handler, keeper, accKeeper
}

func TestHandleCreateAndClaimSwap(t *testing.T) {
	ctx, handler, swapKeeper, accKeeper := setup()
	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(10)

	_, acc1 := testutils.NewAccount(ctx, accKeeper, 10000e8)
	_, acc2 := testutils.NewAccount(ctx, accKeeper, 10000e8)

	randomNumberStr := "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649"
	randomNumber, _ := hex.DecodeString(randomNumberStr)
	timestamp := time.Now().Unix()
	randomNumberHash := CalculateRandomHash(randomNumber, timestamp)

	recipientOtherChain, _ := hex.DecodeString("491e71b619878c083eaf2894718383c7eb15eb17")
	outAmount := sdk.Coin{"BNB", 10000}
	expectedIncome := "10000:BNB"
	heightSpan := int64(1000)

	var msg sdk.Msg
	msg = NewHashTimerLockTransferMsg(acc1.GetAddress(), acc2.GetAddress(), recipientOtherChain, randomNumberHash, timestamp, outAmount, expectedIncome, heightSpan,true)

	result := handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ABCICodeOK)

	AtomicSwapCoinsAcc := accKeeper.GetAccount(ctx, AtomicSwapCoinsAccAddr)
	require.Equal(t, sdk.Coins{outAmount}, AtomicSwapCoinsAcc.GetCoins())

	swap := swapKeeper.QuerySwap(ctx, randomNumberHash)
	require.Equal(t, int64(heightSpan+10), swap.ExpireHeight)

	ctx = ctx.WithBlockHeight(100)

	wrongRandomNumberStr := "62fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649"
	wrongRandomNumber, _ := hex.DecodeString(wrongRandomNumberStr)
	msg = NewClaimHashTimerLockMsg(acc1.GetAddress(), randomNumberHash, wrongRandomNumber)
	result = handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ToABCICode(DefaultCodespace, CodeMismatchedRandomNumber))

	msg = NewRefundLockedAssetMsg(acc2.GetAddress(), randomNumberHash)
	result = handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ToABCICode(DefaultCodespace, CodeRefundUnexpiredSwap))

	msg = NewClaimHashTimerLockMsg(acc1.GetAddress(), randomNumberHash, randomNumber)
	result = handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ABCICodeOK)

	AtomicSwapCoinsAcc = accKeeper.GetAccount(ctx, AtomicSwapCoinsAccAddr)
	require.Equal(t, 0, len(AtomicSwapCoinsAcc.GetCoins()))

	swap = swapKeeper.QuerySwap(ctx, randomNumberHash)
	require.Equal(t, Completed, swap.Status)
}

func TestHandleCreateAndRefundSwap(t *testing.T) {
	ctx, handler, swapKeeper, accKeeper := setup()
	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(10)

	_, acc1 := testutils.NewAccount(ctx, accKeeper, 10000e8)
	_, acc2 := testutils.NewAccount(ctx, accKeeper, 10000e8)

	randomNumberStr := "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649"
	randomNumber, _ := hex.DecodeString(randomNumberStr)
	timestamp := time.Now().Unix()
	randomNumberHash := CalculateRandomHash(randomNumber, timestamp)
	recipientOtherChain, _ := hex.DecodeString("491e71b619878c083eaf2894718383c7eb15eb17")
	outAmount := sdk.Coin{"BNB", 10000}
	expectedIncome := "10000:BNB"
	heightSpan := int64(1000)

	var msg sdk.Msg
	msg = NewHashTimerLockTransferMsg(acc1.GetAddress(), acc2.GetAddress(), recipientOtherChain, randomNumberHash, timestamp, outAmount, expectedIncome, heightSpan, true)

	result := handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ABCICodeOK)

	AtomicSwapCoinsAcc := accKeeper.GetAccount(ctx, AtomicSwapCoinsAccAddr)
	require.Equal(t, sdk.Coins{outAmount}, AtomicSwapCoinsAcc.GetCoins())

	swap := swapKeeper.QuerySwap(ctx, randomNumberHash)
	require.Equal(t, int64(heightSpan+10), swap.ExpireHeight)

	ctx = ctx.WithBlockHeight(2000)

	msg = NewClaimHashTimerLockMsg(acc2.GetAddress(), randomNumberHash, randomNumber)
	result = handler(ctx, msg)
	require.Equal(t, sdk.ToABCICode(DefaultCodespace, CodeClaimExpiredSwap), result.Code)

	AtomicSwapCoinsAcc = accKeeper.GetAccount(ctx, AtomicSwapCoinsAccAddr)
	require.Equal(t, sdk.Coins{outAmount}, AtomicSwapCoinsAcc.GetCoins())

	msg = NewRefundLockedAssetMsg(acc2.GetAddress(), randomNumberHash)
	result = handler(ctx, msg)
	require.Equal(t, sdk.ABCICodeOK, result.Code)

	AtomicSwapCoinsAcc = accKeeper.GetAccount(ctx, AtomicSwapCoinsAccAddr)
	require.Equal(t, 0, len(AtomicSwapCoinsAcc.GetCoins()))

	swap = swapKeeper.QuerySwap(ctx, randomNumberHash)
	require.Equal(t, Expired, swap.Status)
}

func TestHandleCreateAndClaimSwapForSingleChain(t *testing.T) {
	ctx, handler, swapKeeper, accKeeper := setup()
	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(10)

	_, acc1 := testutils.NewAccount(ctx, accKeeper, 10000e8)
	_, acc2 := testutils.NewAccount(ctx, accKeeper, 10000e8)

	acc2Coins := acc2.GetCoins()
	acc2Coins = acc2Coins.Plus(sdk.Coins{sdk.Coin{"ABC", 1000000000000}})
	_ = acc2.SetCoins(acc2Coins)
	accKeeper.SetAccount(ctx, acc2)

	acc1OrignalCoins := accKeeper.GetAccount(ctx, acc1.GetAddress()).GetCoins()
	acc2OrignalCoins := accKeeper.GetAccount(ctx, acc2.GetAddress()).GetCoins()

	randomNumberStr := "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649"
	randomNumber, _ := hex.DecodeString(randomNumberStr)
	timestamp := time.Now().Unix()
	randomNumberHash := CalculateRandomHash(randomNumber, timestamp)

	outAmountBNB := sdk.Coin{"BNB", 10000}
	expectedIncome := "100000000:ABC"
	heightSpan := int64(1000)

	var msg sdk.Msg
	msg = NewHashTimerLockTransferMsg(acc1.GetAddress(), acc2.GetAddress(), nil, randomNumberHash, timestamp, outAmountBNB, expectedIncome, heightSpan, false)

	result := handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ABCICodeOK)

	outAmountABC := sdk.Coin{"ABC", 100000000}
	expectedIncome = "10000:BNB"
	msg = NewHashTimerLockTransferMsg(acc2.GetAddress(), acc1.GetAddress(), nil, randomNumberHash, timestamp, outAmountABC, expectedIncome, heightSpan, false)

	result = handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ABCICodeOK)

	AtomicSwapCoinsAcc := accKeeper.GetAccount(ctx, AtomicSwapCoinsAccAddr)
	require.Equal(t, sdk.Coins{outAmountBNB}.Plus(sdk.Coins{outAmountABC}).Sort(), AtomicSwapCoinsAcc.GetCoins())

	swap := swapKeeper.QuerySwap(ctx, randomNumberHash)
	require.Equal(t, int64(heightSpan+10), swap.ExpireHeight)

	ctx = ctx.WithBlockHeight(20)

	wrongRandomNumberStr := "62fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649"
	wrongRandomNumber, _ := hex.DecodeString(wrongRandomNumberStr)
	msg = NewClaimHashTimerLockMsg(acc1.GetAddress(), randomNumberHash, wrongRandomNumber)
	result = handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ToABCICode(DefaultCodespace, CodeMismatchedRandomNumber))

	msg = NewRefundLockedAssetMsg(acc2.GetAddress(), randomNumberHash)
	result = handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ToABCICode(DefaultCodespace, CodeRefundUnexpiredSwap))

	msg = NewClaimHashTimerLockMsg(acc1.GetAddress(), randomNumberHash, randomNumber)
	result = handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ABCICodeOK)

	AtomicSwapCoinsAcc = accKeeper.GetAccount(ctx, AtomicSwapCoinsAccAddr)
	require.Equal(t, 0, len(AtomicSwapCoinsAcc.GetCoins()))

	swap = swapKeeper.QuerySwap(ctx, randomNumberHash)
	require.Equal(t, Completed, swap.Status)

	acc1Acc := accKeeper.GetAccount(ctx, acc1.GetAddress())
	require.Equal(t, outAmountABC.Amount, acc1Acc.GetCoins().AmountOf("ABC"))
	require.Equal(t, outAmountBNB.Amount, acc1OrignalCoins.AmountOf("BNB")-acc1Acc.GetCoins().AmountOf("BNB"))

	acc2Acc := accKeeper.GetAccount(ctx, acc2.GetAddress())
	require.Equal(t, outAmountBNB.Amount, acc2Acc.GetCoins().AmountOf("BNB")-acc2OrignalCoins.AmountOf("BNB"))
	require.Equal(t, outAmountABC.Amount, acc2OrignalCoins.AmountOf("ABC")-acc2Acc.GetCoins().AmountOf("ABC"))
}

func TestHandleCreateAndRefundSwapForSingleChain(t *testing.T) {
	ctx, handler, swapKeeper, accKeeper := setup()
	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(10)

	_, acc1 := testutils.NewAccount(ctx, accKeeper, 10000e8)
	_, acc2 := testutils.NewAccount(ctx, accKeeper, 10000e8)

	acc2Coins := acc2.GetCoins()
	acc2Coins = acc2Coins.Plus(sdk.Coins{sdk.Coin{"ABC", 1000000000000}})
	_ = acc2.SetCoins(acc2Coins)
	accKeeper.SetAccount(ctx, acc2)

	acc1OrignalCoins := accKeeper.GetAccount(ctx, acc1.GetAddress()).GetCoins()
	acc2OrignalCoins := accKeeper.GetAccount(ctx, acc2.GetAddress()).GetCoins()

	randomNumberStr := "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649"
	randomNumber, _ := hex.DecodeString(randomNumberStr)
	timestamp := time.Now().Unix()
	randomNumberHash := CalculateRandomHash(randomNumber, timestamp)

	outAmountBNB := sdk.Coin{"BNB", 10000}
	expectedIncome := "100000000:ABC"
	heightSpan := int64(1000)

	var msg sdk.Msg
	msg = NewHashTimerLockTransferMsg(acc1.GetAddress(), acc2.GetAddress(), nil, randomNumberHash, timestamp, outAmountBNB, expectedIncome, heightSpan, false)

	result := handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ABCICodeOK)

	outAmountABC := sdk.Coin{"ABC", 100000000}
	expectedIncome = "10000:BNB"
	msg = NewHashTimerLockTransferMsg(acc2.GetAddress(), acc1.GetAddress(), nil, randomNumberHash, timestamp, outAmountABC, expectedIncome, heightSpan, false)

	result = handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ABCICodeOK)

	AtomicSwapCoinsAcc := accKeeper.GetAccount(ctx, AtomicSwapCoinsAccAddr)
	require.Equal(t, sdk.Coins{outAmountBNB}.Plus(sdk.Coins{outAmountABC}).Sort(), AtomicSwapCoinsAcc.GetCoins())

	swap := swapKeeper.QuerySwap(ctx, randomNumberHash)
	require.Equal(t, int64(heightSpan+10), swap.ExpireHeight)

	ctx = ctx.WithBlockHeight(2000)

	msg = NewClaimHashTimerLockMsg(acc2.GetAddress(), randomNumberHash, randomNumber)
	result = handler(ctx, msg)
	require.Equal(t, sdk.ToABCICode(DefaultCodespace, CodeClaimExpiredSwap), result.Code)

	AtomicSwapCoinsAcc = accKeeper.GetAccount(ctx, AtomicSwapCoinsAccAddr)
	require.Equal(t, sdk.Coins{outAmountBNB}.Plus(sdk.Coins{outAmountABC}).Sort(), AtomicSwapCoinsAcc.GetCoins())

	msg = NewRefundLockedAssetMsg(acc2.GetAddress(), randomNumberHash)
	result = handler(ctx, msg)
	require.Equal(t, sdk.ABCICodeOK, result.Code)

	AtomicSwapCoinsAcc = accKeeper.GetAccount(ctx, AtomicSwapCoinsAccAddr)
	require.Equal(t, 0, len(AtomicSwapCoinsAcc.GetCoins()))

	swap = swapKeeper.QuerySwap(ctx, randomNumberHash)
	require.Equal(t, Expired, swap.Status)

	acc1Acc := accKeeper.GetAccount(ctx, acc1.GetAddress())
	require.Equal(t, acc1OrignalCoins, acc1Acc.GetCoins())

	acc2Acc := accKeeper.GetAccount(ctx, acc2.GetAddress())
	require.Equal(t, acc2OrignalCoins, acc2Acc.GetCoins())
}
