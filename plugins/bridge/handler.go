package bridge

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
	"strings"

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
		case TransferOutTimeoutMsg:
			return handleTransferOutTimeoutMsg(ctx, keeper, msg)
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

	claim, err := types.CreateOracleClaimFromTransferMsg(msg)
	if err != nil {
		return err.Result()
	}

	_, err = bridgeKeeper.ProcessTransferClaim(ctx, claim)
	if err != nil {
		return err.Result()
	}

	tags := sdk.NewTags(
		types.Action, types.ActionTransferIn,
	)

	return sdk.Result{Tags: tags}
}

func handleTransferOutTimeoutMsg(ctx sdk.Context, bridgeKeeper Keeper, msg TransferOutTimeoutMsg) sdk.Result {
	currentSequence := bridgeKeeper.GetCurrentSequence(ctx, types.KeyTransferOutTimeoutSequence)
	if msg.Sequence != currentSequence {
		return types.ErrInvalidSequence(fmt.Sprintf("current sequence is %d", currentSequence)).Result()
	}

	claim, err := types.CreateOracleClaimFromTransferOutTimeoutMsg(msg)
	if err != nil {
		return err.Result()
	}

	_, err = bridgeKeeper.ProcessTimeoutClaim(ctx, claim)
	if err != nil {
		return err.Result()
	}

	tags := sdk.NewTags(
		types.Action, types.ActionTimeout,
	)

	return sdk.Result{Tags: tags}
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
		types.Action, types.ActionUpdateBind,
	)

	return sdk.Result{Tags: tags}
}

func handleBindMsg(ctx sdk.Context, keeper Keeper, msg BindMsg) sdk.Result {
	if msg.ExpireTime < types.MinBindExpireTimeGap {
		return types.ErrInvalidExpireTime(fmt.Sprintf("expire time %d is less than minimum expire time gap %d )",
			msg.ExpireTime, types.MinBindExpireTimeGap)).Result()
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

	if !token.IsOwner(msg.From) {
		return sdk.ErrUnauthorized(fmt.Sprintf("only the owner can bind token %s", msg.Symbol)).Result()
	}

	peggyAmount := sdk.Coins{sdk.Coin{Denom: symbol, Amount: msg.Amount}}
	relayFee := sdk.Coins{sdk.Coin{Denom: cmmtypes.NativeTokenSymbol, Amount: types.RelayReward}}
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
	calibratedRelayFee := sdk.NewInt(types.RelayReward).Mul(sdk.NewIntWithDecimal(1, int(18-cmmtypes.TokenDecimals)))

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

	expireTime := msg.ExpireTime+ctx.BlockHeader().Time.Unix()
	bindPackage, err := types.SerializeBindPackage(symbol, msg.ContractAddress[:],
		calibratedTotalSupply, calibratedAmount, expireTime, calibratedRelayFee)
	if err != nil {
		return types.ErrSerializePackageFailed(err.Error()).Result()
	}

	bindChannelId, err := sdk.GetChannelID(types.BindChannelName)
	if err != nil {
		return types.ErrGetChannelIdFailed(err.Error()).Result()
	}

	sdkErr = keeper.IbcKeeper.CreateIBCPackage(ctx, sdk.CrossChainID(keeper.DestChainId), bindChannelId, bindPackage)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	tags := sdk.NewTags(
		types.Action, types.ActionBind,
		types.ExpireTime, []byte(strconv.Itoa(int(expireTime))),
	)

	return sdk.Result{Tags: tags}
}

func handleTransferOutMsg(ctx sdk.Context, keeper Keeper, msg TransferOutMsg) sdk.Result {
	if msg.ExpireTime < types.MinTransferOutExpireTimeGap {
		return types.ErrInvalidExpireTime(fmt.Sprintf("expire time %d is less than minimum expire time gap %d )",
			msg.ExpireTime, types.MinTransferOutExpireTimeGap)).Result()
	}

	symbol := strings.ToUpper(msg.Amount.Denom)
	token, err := keeper.TokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", symbol)).Result()
	}

	if token.ContractAddress == "" {
		return types.ErrTokenNotBound(fmt.Sprintf("token %s is not bound", symbol)).Result()
	}

	transferAmount := sdk.Coins{msg.Amount}.Plus(sdk.Coins{sdk.Coin{Denom: cmmtypes.NativeTokenSymbol, Amount: types.RelayReward}})
	_, cErr := keeper.BankKeeper.SendCoins(ctx, msg.From, types.PegAccount, transferAmount)
	if cErr != nil {
		return cErr.Result()
	}

	var calibratedAmount sdk.Int
	if token.ContractDecimal >= cmmtypes.TokenDecimals {
		calibratedAmount = sdk.NewInt(msg.Amount.Amount).Mul(sdk.NewIntWithDecimal(1, int(token.ContractDecimal-cmmtypes.TokenDecimals)))
	} else {
		decimals := sdk.NewIntWithDecimal(1, int(cmmtypes.TokenDecimals-token.ContractDecimal))
		if !sdk.NewInt(msg.Amount.Amount).Mod(decimals).IsZero() {
			return types.ErrInvalidAmount("can't calibrate transfer amount to the amount of ERC20").Result()
		}
		calibratedAmount = sdk.NewInt(msg.Amount.Amount).Div(decimals)
	}
	calibratedRelayFee := sdk.NewInt(types.RelayReward).Mul(sdk.NewIntWithDecimal(1, int(18-cmmtypes.TokenDecimals)))

	contractAddr := types.NewEthereumAddress(token.ContractAddress)
	expireTime := msg.ExpireTime+ctx.BlockHeader().Time.Unix()
	transferPackage, err := types.SerializeTransferOutPackage(symbol, contractAddr[:], msg.From.Bytes(), msg.To[:],
		calibratedAmount, expireTime, calibratedRelayFee)
	if err != nil {
		return types.ErrSerializePackageFailed(err.Error()).Result()
	}

	transferChannelId, err := sdk.GetChannelID(types.TransferOutChannelName)
	if err != nil {
		return types.ErrGetChannelIdFailed(err.Error()).Result()
	}

	sdkErr := keeper.IbcKeeper.CreateIBCPackage(ctx, sdk.CrossChainID(keeper.DestChainId), transferChannelId, transferPackage)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	tags := sdk.NewTags(
		types.Action, types.ActionTransferOut,
		types.ExpireTime, []byte(strconv.Itoa(int(expireTime))),
	)

	return sdk.Result{Tags: tags}
}
