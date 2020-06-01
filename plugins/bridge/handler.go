package bridge

import (
	"fmt"
	"strings"
	"time"

	"github.com/binance-chain/node/common/log"

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
		case UnbindMsg:
			return handleUnbindMsg(ctx, keeper, msg)
		default:
			errMsg := "Unrecognized bridge msg type"
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleUnbindMsg(ctx sdk.Context, keeper Keeper, msg UnbindMsg) sdk.Result {
	// check is native symbol
	if msg.Symbol == cmmtypes.NativeTokenSymbol {
		return types.ErrInvalidSymbol("can not unbind native symbol").Result()
	}

	token, err := keeper.TokenMapper.GetToken(ctx, msg.Symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", msg.Symbol)).Result()
	}

	if token.ContractAddress == "" {
		return types.ErrTokenBound(fmt.Sprintf("token %s is not bound", msg.Symbol)).Result()
	}

	if !token.IsOwner(msg.From) {
		return sdk.ErrUnauthorized(fmt.Sprintf("only the owner can unbind token %s", msg.Symbol)).Result()
	}

	unbindPackage := types.BindPackage{
		BindType:        types.BindTypeUnbind,
		Bep2TokenSymbol: msg.Symbol,
	}
	serializedPackage, err := types.SerializeBindPackage(&unbindPackage)
	if err != nil {
		log.With("module", "bridge").Error("serialize unbind package error", "err", err.Error())
		return types.ErrSerializePackageFailed(err.Error()).Result()
	}

	_, sdkErr := keeper.IbcKeeper.CreateIBCPackage(ctx, keeper.DestChainId, types.BindChannel, serializedPackage)
	if sdkErr != nil {
		log.With("module", "bridge").Error("create unbind ibc package error", "err", sdkErr.Error())
		return sdkErr.Result()
	}

	err = keeper.TokenMapper.UpdateBind(ctx, msg.Symbol, "", 0)
	if err != nil {
		log.With("module", "bridge").Error("update token info error", "err", err.Error())
		return sdk.ErrInternal(fmt.Sprintf("update token error, err=%s", err.Error())).Result()
	}

	log.With("module", "bridge").Info("unbind token success", "symbol", msg.Symbol, "contract_addr", token.ContractAddress)

	return sdk.Result{}
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

	// if token owner has bound token before, decimals should be the same as the original when contract address is the same.
	existsContractDecimals := keeper.GetContractDecimals(ctx, msg.ContractAddress)
	if existsContractDecimals >= 0 && existsContractDecimals != msg.ContractDecimals {
		return types.ErrInvalidDecimals(fmt.Sprintf("decimals should be %d", existsContractDecimals)).Result()
	}

	tokenAmount := keeper.BankKeeper.GetCoins(ctx, types.PegAccount).AmountOf(symbol)
	if msg.Amount < tokenAmount {
		return sdk.ErrInvalidCoins(fmt.Sprintf("bind amount should be no less than %d", tokenAmount)).Result()
	}

	peggyAmount := sdk.Coins{
		sdk.Coin{Denom: symbol, Amount: msg.Amount - tokenAmount},
	}
	relayFee, sdkErr := types.GetFee(types.BindRelayFeeName)
	if sdkErr != nil {
		return sdkErr.Result()
	}
	transferAmount := peggyAmount.Plus(relayFee.Tokens)

	bscTotalSupply, sdkErr := types.ConvertBCAmountToBSCAmount(msg.ContractDecimals, token.TotalSupply.ToInt64())
	if sdkErr != nil {
		return sdkErr.Result()
	}
	bscAmount, sdkErr := types.ConvertBCAmountToBSCAmount(msg.ContractDecimals, msg.Amount)
	if sdkErr != nil {
		return sdkErr.Result()
	}
	bscRelayFee, sdkErr := types.ConvertBCAmountToBSCAmount(types.BSCBNBDecimals, relayFee.Tokens.AmountOf(cmmtypes.NativeTokenSymbol))
	if sdkErr != nil {
		return sdkErr.Result()
	}

	_, sdkErr = keeper.BankKeeper.SendCoins(ctx, msg.From, types.PegAccount, transferAmount)
	if sdkErr != nil {
		log.With("module", "bridge").Error("send coins error", "err", sdkErr.Error())
		return sdkErr.Result()
	}

	bindRequest := types.BindRequest{
		From:             msg.From,
		Symbol:           symbol,
		Amount:           msg.Amount,
		DeductedAmount:   msg.Amount - tokenAmount,
		ContractAddress:  msg.ContractAddress,
		ContractDecimals: msg.ContractDecimals,
		ExpireTime:       msg.ExpireTime,
	}
	sdkErr = keeper.CreateBindRequest(ctx, bindRequest)
	if sdkErr != nil {
		log.With("module", "bridge").Error("create bind request error", "err", sdkErr.Error())
		return sdkErr.Result()
	}

	bindPackage := types.BindPackage{
		BindType:        types.BindTypeBind,
		Bep2TokenSymbol: symbol,
		ContractAddr:    msg.ContractAddress[:],
		TotalSupply:     bscTotalSupply,
		PeggyAmount:     bscAmount,
		Decimals:        msg.ContractDecimals,
		ExpireTime:      msg.ExpireTime,
		RelayReward:     bscRelayFee,
	}
	serializedPackage, err := types.SerializeBindPackage(&bindPackage)
	if err != nil {
		log.With("module", "bridge").Error("serialize bind package error", "err", err.Error())
		return types.ErrSerializePackageFailed(err.Error()).Result()
	}

	_, sdkErr = keeper.IbcKeeper.CreateIBCPackage(ctx, keeper.DestChainId, types.BindChannel, serializedPackage)
	if sdkErr != nil {
		log.With("module", "bridge").Error("create bind ibc package error", "err", sdkErr.Error())
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

	if token.ContractAddress == "" {
		return types.ErrTokenNotBound(fmt.Sprintf("token %s is not bound", symbol)).Result()
	}

	fee, sdkErr := types.GetFee(types.TransferOutFeeName)
	if sdkErr != nil {
		log.With("module", "bridge").Error("get transfer out fee error", "err", sdkErr.Error())
		return sdkErr.Result()
	}
	transferAmount := sdk.Coins{msg.Amount}.Plus(fee.Tokens)

	bscTransferAmount, sdkErr := types.ConvertBCAmountToBSCAmount(token.ContractDecimals, msg.Amount.Amount)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	bscRelayFee, sdkErr := types.ConvertBCAmountToBSCAmount(types.BSCBNBDecimals, fee.Tokens.AmountOf(cmmtypes.NativeTokenSymbol))
	if sdkErr != nil {
		return sdkErr.Result()
	}

	_, sdkErr = keeper.BankKeeper.SendCoins(ctx, msg.From, types.PegAccount, transferAmount)
	if sdkErr != nil {
		log.With("module", "bridge").Error("send coins error", "err", sdkErr.Error())
		return sdkErr.Result()
	}

	contractAddr := types.NewSmartChainAddress(token.ContractAddress)
	transferPackage, err := types.SerializeTransferOutPackage(symbol, contractAddr[:], msg.From.Bytes(), msg.To[:],
		bscTransferAmount, msg.ExpireTime, bscRelayFee)
	if err != nil {
		log.With("module", "bridge").Error("serialize transfer out package error", "err", err.Error())
		return types.ErrSerializePackageFailed(err.Error()).Result()
	}

	_, sdkErr = keeper.IbcKeeper.CreateIBCPackage(ctx, keeper.DestChainId, types.TransferOutChannel, transferPackage)
	if sdkErr != nil {
		log.With("module", "bridge").Error("create transfer out ibc package error", "err", sdkErr.Error())
		return sdkErr.Result()
	}

	if ctx.IsDeliverTx() {
		keeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, msg.From})
	}
	return sdk.Result{}
}
