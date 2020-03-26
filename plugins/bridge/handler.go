package bridge

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmmtypes "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/bridge/types"
)

func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case TransferInMsg:
			return handleTransferInMsg(ctx, keeper, msg)
		case TransferOutMsg:
			return handleTransferOutMsg(ctx, keeper, msg)
		case UpdateBindMsg:
			return handleUpdateBindMsg(ctx, keeper, msg)
		case BindMsg:
			return handleBindMsg(ctx, keeper, msg)
		case UpdateTransferOutMsg:
			return handleUpdateTransferOutMsg(ctx, keeper, msg)
		default:
			errMsg := "Unrecognized bridge msg type"
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleTransferInMsg(ctx sdk.Context, bridgeKeeper Keeper, msg TransferInMsg) sdk.Result {
	if msg.RelayFee.Denom != cmmtypes.NativeTokenSymbol {
		return types.ErrInvalidSymbol(fmt.Sprintf("relay fee should be native token(%s)", cmmtypes.NativeTokenSymbol)).Result()
	}

	currentSequence := bridgeKeeper.GetCurrentSequence(ctx, types.KeyCurrentTransferInSequence)
	if msg.Sequence != currentSequence {
		return types.ErrInvalidSequence(fmt.Sprintf("current sequence is %d", currentSequence)).Result()
	}

	claim, err := types.CreateOracleClaimFromTransferInMsg(msg)
	if err != nil {
		return err.Result()
	}

	tags, err := bridgeKeeper.ProcessTransferInClaim(ctx, claim)
	if err != nil {
		return err.Result()
	}

	resultTags := sdk.NewTags(
		types.TransferInSequence, []byte(strconv.Itoa(int(msg.Sequence))),
	)
	resultTags = resultTags.AppendTags(tags)
	return sdk.Result{Tags: resultTags}
}

func handleUpdateTransferOutMsg(ctx sdk.Context, bridgeKeeper Keeper, msg UpdateTransferOutMsg) sdk.Result {
	currentSequence := bridgeKeeper.GetCurrentSequence(ctx, types.KeyUpdateTransferOutSequence)
	if msg.Sequence != currentSequence {
		return types.ErrInvalidSequence(fmt.Sprintf("current sequence is %d", currentSequence)).Result()
	}

	claim, err := types.CreateOracleClaimFromUpdateTransferOutMsg(msg)
	if err != nil {
		return err.Result()
	}

	_, tags, err := bridgeKeeper.ProcessUpdateTransferOutClaim(ctx, claim)
	if err != nil {
		return err.Result()
	}

	resultTags := sdk.NewTags(
		types.TransferOutRefundSequence, []byte(strconv.Itoa(int(msg.Sequence))),
	)

	resultTags = resultTags.AppendTags(tags)

	return sdk.Result{Tags: resultTags}
}

func handleUpdateBindMsg(ctx sdk.Context, keeper Keeper, msg UpdateBindMsg) sdk.Result {
	currentSequence := keeper.GetCurrentSequence(ctx, types.KeyUpdateBindSequence)
	if msg.Sequence != currentSequence {
		return types.ErrInvalidSequence(fmt.Sprintf("current sequence is %d", currentSequence)).Result()
	}

	if _, err := keeper.TokenMapper.GetToken(ctx, msg.Symbol); err != nil {
		return types.ErrInvalidSymbol(fmt.Sprintf("token %s is not existing", msg.Symbol)).Result()
	}

	claim, err := types.CreateOracleClaimFromUpdateBindMsg(msg)
	if err != nil {
		return err.Result()
	}

	_, err = keeper.ProcessUpdateBindClaim(ctx, claim)
	if err != nil {
		return err.Result()
	}

	tags := sdk.NewTags(
		types.UpdateBindSequence, []byte(strconv.Itoa(int(msg.Sequence))),
	)

	return sdk.Result{Tags: tags}
}

func handleBindMsg(ctx sdk.Context, keeper Keeper, msg BindMsg) sdk.Result {
	if !time.Unix(msg.ExpireTime, 0).After(ctx.BlockHeader().Time.Add(types.MinBindExpireTimeGap)) {
		return types.ErrInvalidExpireTime(fmt.Sprintf("expire time should be %d seconds after now(%s)",
			types.MinBindExpireTimeGap, ctx.BlockHeader().Time.UTC().String())).Result()
	}

	symbol := strings.ToUpper(msg.Symbol)

	// check is native symbol
	if symbol == cmmtypes.NativeTokenSymbol {
		return types.ErrInvalidSymbol("can not bind native symbol").Result()
	}

	token, err := keeper.TokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", msg.Symbol)).Result()
	}

	if token.ContractAddress != "" {
		return types.ErrTokenBound(fmt.Sprintf("token %s is already bound", symbol)).Result()
	}

	if !token.IsOwner(msg.From) {
		return sdk.ErrUnauthorized(fmt.Sprintf("only the owner can bind token %s", msg.Symbol)).Result()
	}

	peggyAmount := sdk.Coins{sdk.Coin{Denom: symbol, Amount: msg.Amount}}
	relayFee := sdk.Coins{sdk.Coin{Denom: cmmtypes.NativeTokenSymbol, Amount: types.RelayFee}}
	transferAmount := peggyAmount.Plus(relayFee)

	_, sdkErr := keeper.BankKeeper.SendCoins(ctx, msg.From, types.PegAccount, transferAmount)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	var calibratedTotalSupply sdk.Int
	var calibratedAmount sdk.Int
	if msg.ContractDecimals >= cmmtypes.TokenDecimals {
		decimals := sdk.NewIntWithDecimal(1, int(msg.ContractDecimals-cmmtypes.TokenDecimals))
		calibratedTotalSupply = sdk.NewInt(token.TotalSupply.ToInt64()).Mul(decimals)
		calibratedAmount = sdk.NewInt(msg.Amount).Mul(decimals)
	} else {
		decimals := sdk.NewIntWithDecimal(1, int(cmmtypes.TokenDecimals-msg.ContractDecimals))
		if !sdk.NewInt(token.TotalSupply.ToInt64()).Mod(decimals).IsZero() || !sdk.NewInt(msg.Amount).Mod(decimals).IsZero() {
			return types.ErrInvalidAmount(fmt.Sprintf("can't calibrate bep2(decimals: 8) amount to ERC20(decimals: %d) amount", msg.ContractDecimals)).Result()
		}
		calibratedTotalSupply = sdk.NewInt(token.TotalSupply.ToInt64()).Div(decimals)
		calibratedAmount = sdk.NewInt(msg.Amount).Div(decimals)
	}
	calibratedRelayFee := sdk.NewInt(types.RelayFee).Mul(sdk.NewIntWithDecimal(1, int(18-cmmtypes.TokenDecimals)))

	bindRequest := types.BindRequest{
		From:             msg.From,
		Symbol:           msg.Symbol,
		Amount:           calibratedAmount,
		ContractAddress:  msg.ContractAddress,
		ContractDecimals: msg.ContractDecimals,
		ExpireTime:       msg.ExpireTime,
	}
	sdkErr = keeper.CreateBindRequest(ctx, bindRequest)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	bindPackage, err := types.SerializeBindPackage(symbol, msg.ContractAddress[:],
		calibratedTotalSupply, calibratedAmount, msg.ExpireTime, calibratedRelayFee)
	if err != nil {
		return types.ErrSerializePackageFailed(err.Error()).Result()
	}

	bindSequence, sdkErr := keeper.IbcKeeper.CreateIBCPackage(ctx, cmmtypes.BSCChain, cmmtypes.BindChannel, bindPackage)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	tags := sdk.NewTags(
		types.BindSequence, []byte(strconv.Itoa(int(bindSequence))),
		types.ExpireTime, []byte(strconv.Itoa(int(msg.ExpireTime))),
	)

	return sdk.Result{Tags: tags}
}

func handleTransferOutMsg(ctx sdk.Context, keeper Keeper, msg TransferOutMsg) sdk.Result {
	if !time.Unix(msg.ExpireTime, 0).After(ctx.BlockHeader().Time.Add(types.MinTransferOutExpireTimeGap)) {
		return types.ErrInvalidExpireTime(fmt.Sprintf("expire time should be %d seconds after now(%s)",
			types.MinTransferOutExpireTimeGap, ctx.BlockHeader().Time.UTC().String())).Result()
	}

	symbol := strings.ToUpper(msg.Amount.Denom)
	token, err := keeper.TokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", symbol)).Result()
	}

	if token.ContractAddress == "" {
		return types.ErrTokenNotBound(fmt.Sprintf("token %s is not bound", symbol)).Result()
	}

	transferAmount := sdk.Coins{msg.Amount}.Plus(sdk.Coins{sdk.Coin{Denom: cmmtypes.NativeTokenSymbol, Amount: types.RelayFee}})
	_, cErr := keeper.BankKeeper.SendCoins(ctx, msg.From, types.PegAccount, transferAmount)
	if cErr != nil {
		return cErr.Result()
	}

	var calibratedAmount sdk.Int
	if token.ContractDecimals >= cmmtypes.TokenDecimals {
		calibratedAmount = sdk.NewInt(msg.Amount.Amount).Mul(sdk.NewIntWithDecimal(1, int(token.ContractDecimals-cmmtypes.TokenDecimals)))
	} else {
		decimals := sdk.NewIntWithDecimal(1, int(cmmtypes.TokenDecimals-token.ContractDecimals))
		if !sdk.NewInt(msg.Amount.Amount).Mod(decimals).IsZero() {
			return types.ErrInvalidAmount("can't calibrate transfer amount to the amount of ERC20").Result()
		}
		calibratedAmount = sdk.NewInt(msg.Amount.Amount).Div(decimals)
	}
	calibratedRelayFee := sdk.NewInt(types.RelayFee).Mul(sdk.NewIntWithDecimal(1, int(18-cmmtypes.TokenDecimals)))

	contractAddr := types.NewEthereumAddress(token.ContractAddress)
	transferPackage, err := types.SerializeTransferOutPackage(symbol, contractAddr[:], msg.From.Bytes(), msg.To[:],
		calibratedAmount, msg.ExpireTime, calibratedRelayFee)
	if err != nil {
		return types.ErrSerializePackageFailed(err.Error()).Result()
	}

	transferOutSequence, sdkErr := keeper.IbcKeeper.CreateIBCPackage(ctx, cmmtypes.BSCChain, cmmtypes.TransferOutChannel, transferPackage)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	tags := sdk.NewTags(
		types.TransferOutSequence, []byte(strconv.Itoa(int(transferOutSequence))),
		types.ExpireTime, []byte(strconv.Itoa(int(msg.ExpireTime))),
	)

	return sdk.Result{Tags: tags}
}
