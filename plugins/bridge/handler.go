package bridge

import (
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmmtypes "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/bridge/types"
)

func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case TransferOutMsg:
			return handleTransferOutMsg(ctx, keeper, msg)
		case BindMsg:
			return handleBindMsg(ctx, keeper, msg)
		default:
			errMsg := "Unrecognized bridge msg type"
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
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

	if token.GetContractAddress() != "" {
		return types.ErrTokenBound(fmt.Sprintf("token %s is already bound", symbol)).Result()
	}

	if !token.IsOwner(msg.From) {
		return sdk.ErrUnauthorized(fmt.Sprintf("only the owner can bind token %s", msg.Symbol)).Result()
	}

	peggyAmount := sdk.Coins{sdk.Coin{Denom: symbol, Amount: msg.Amount}}
	relayFee, sdkErr := types.GetFee(types.BindRelayFeeName)
	if sdkErr != nil {
		return sdkErr.Result()
	}
	transferAmount := peggyAmount.Plus(relayFee.Tokens)

	_, sdkErr = keeper.BankKeeper.SendCoins(ctx, msg.From, types.PegAccount, transferAmount)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	var calibratedTotalSupply sdk.Int
	var calibratedAmount sdk.Int
	if msg.ContractDecimals >= cmmtypes.TokenDecimals {
		decimals := sdk.NewIntWithDecimal(1, int(msg.ContractDecimals-cmmtypes.TokenDecimals))
		calibratedTotalSupply = sdk.NewInt(token.GetTotalSupply().ToInt64()).Mul(decimals)
		calibratedAmount = sdk.NewInt(msg.Amount).Mul(decimals)
	} else {
		decimals := sdk.NewIntWithDecimal(1, int(cmmtypes.TokenDecimals-msg.ContractDecimals))
		if !sdk.NewInt(token.GetTotalSupply().ToInt64()).Mod(decimals).IsZero() || !sdk.NewInt(msg.Amount).Mod(decimals).IsZero() {
			return types.ErrInvalidAmount(fmt.Sprintf("can't convert bep2(decimals: 8) amount to ERC20(decimals: %d) amount", msg.ContractDecimals)).Result()
		}
		calibratedTotalSupply = sdk.NewInt(token.GetTotalSupply().ToInt64()).Div(decimals)
		calibratedAmount = sdk.NewInt(msg.Amount).Div(decimals)
	}
	calibratedRelayFee := sdk.NewInt(relayFee.Tokens.AmountOf(cmmtypes.NativeTokenSymbol)).Mul(sdk.NewIntWithDecimal(1, int(18-cmmtypes.TokenDecimals)))

	bindRequest := types.BindRequest{
		From:             msg.From,
		Symbol:           symbol,
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
		calibratedTotalSupply, calibratedAmount, msg.ContractDecimals, msg.ExpireTime, calibratedRelayFee)
	if err != nil {
		return types.ErrSerializePackageFailed(err.Error()).Result()
	}

	_, sdkErr = keeper.IbcKeeper.CreateIBCPackage(ctx, keeper.DestChainId, types.BindChannel, bindPackage)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	if ctx.IsDeliverTx() {
		keeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, msg.From})
	}
	return sdk.Result{}
}

func handleTransferOutMsg(ctx sdk.Context, keeper Keeper, msg TransferOutMsg) sdk.Result {
	if !time.Unix(msg.ExpireTime, 0).After(ctx.BlockHeader().Time.Add(types.MinTransferOutExpireTimeGap)) {
		return types.ErrInvalidExpireTime(fmt.Sprintf("expire time should be %d seconds after now(%s)",
			types.MinTransferOutExpireTimeGap, ctx.BlockHeader().Time.UTC().String())).Result()
	}

	symbol := msg.Amount.Denom
	token, err := keeper.TokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", symbol)).Result()
	}

	if token.GetContractAddress() == "" {
		return types.ErrTokenNotBound(fmt.Sprintf("token %s is not bound", symbol)).Result()
	}

	fee, sdkErr := types.GetFee(types.TransferOutFeeName)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	transferAmount := sdk.Coins{msg.Amount}.Plus(fee.Tokens)
	_, cErr := keeper.BankKeeper.SendCoins(ctx, msg.From, types.PegAccount, transferAmount)
	if cErr != nil {
		return cErr.Result()
	}

	var calibratedAmount sdk.Int
	if token.GetContractDecimals() >= cmmtypes.TokenDecimals {
		calibratedAmount = sdk.NewInt(msg.Amount.Amount).Mul(sdk.NewIntWithDecimal(1, int(token.GetContractDecimals()-cmmtypes.TokenDecimals)))
	} else {
		decimals := sdk.NewIntWithDecimal(1, int(cmmtypes.TokenDecimals-token.GetContractDecimals()))
		if !sdk.NewInt(msg.Amount.Amount).Mod(decimals).IsZero() {
			return types.ErrInvalidAmount(fmt.Sprintf("can't convert bep2(decimals: 8) amount %d to ERC20(decimals: %d) amount", msg.Amount.Amount, token.GetContractDecimals())).Result()
		}
		calibratedAmount = sdk.NewInt(msg.Amount.Amount).Div(decimals)
	}
	calibratedRelayFee := sdk.NewInt(fee.Tokens.AmountOf(cmmtypes.NativeTokenSymbol)).Mul(sdk.NewIntWithDecimal(1, int(18-cmmtypes.TokenDecimals)))

	contractAddr := types.NewSmartChainAddress(token.GetContractAddress())
	transferPackage, err := types.SerializeTransferOutPackage(symbol, contractAddr[:], msg.From.Bytes(), msg.To[:],
		calibratedAmount, msg.ExpireTime, calibratedRelayFee)
	if err != nil {
		return types.ErrSerializePackageFailed(err.Error()).Result()
	}

	_, sdkErr = keeper.IbcKeeper.CreateIBCPackage(ctx, keeper.DestChainId, types.TransferOutChannel, transferPackage)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	if ctx.IsDeliverTx() {
		keeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, msg.From})
	}
	return sdk.Result{}
}
