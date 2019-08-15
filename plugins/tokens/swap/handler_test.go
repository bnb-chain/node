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

	randomNumberHash, _ := hex.DecodeString("be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167")
	timestamp := int64(1564471835)
	recipientOtherChain, _ := hex.DecodeString("491e71b619878c083eaf2894718383c7eb15eb17")
	outAmount := sdk.Coin{"BNB", 10000}
	inAmountOtherChain := int64(10000)
	heightSpan := int64(1000)

	var msg sdk.Msg
	msg = NewHashTimerLockTransferMsg(acc1.GetAddress(), acc2.GetAddress(), recipientOtherChain, randomNumberHash, timestamp, outAmount, inAmountOtherChain, heightSpan)

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

	randomNumberStr := "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649"
	randomNumber, _ := hex.DecodeString(randomNumberStr)
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

	randomNumberHash, _ := hex.DecodeString("be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167")
	timestamp := int64(1564471835)
	recipientOtherChain, _ := hex.DecodeString("491e71b619878c083eaf2894718383c7eb15eb17")
	outAmount := sdk.Coin{"BNB", 10000}
	inAmountOtherChain := int64(10000)
	heightSpan := int64(1000)

	var msg sdk.Msg
	msg = NewHashTimerLockTransferMsg(acc1.GetAddress(), acc2.GetAddress(), recipientOtherChain, randomNumberHash, timestamp, outAmount, inAmountOtherChain, heightSpan)

	result := handler(ctx, msg)
	require.Equal(t, result.Code, sdk.ABCICodeOK)

	AtomicSwapCoinsAcc := accKeeper.GetAccount(ctx, AtomicSwapCoinsAccAddr)
	require.Equal(t, sdk.Coins{outAmount}, AtomicSwapCoinsAcc.GetCoins())

	swap := swapKeeper.QuerySwap(ctx, randomNumberHash)
	require.Equal(t, int64(heightSpan+10), swap.ExpireHeight)

	ctx = ctx.WithBlockHeight(2000)

	randomNumberStr := "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649"
	randomNumber, _ := hex.DecodeString(randomNumberStr)
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
