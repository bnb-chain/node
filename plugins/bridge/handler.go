package bridge

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/bsc/rlp"
	"github.com/cosmos/cosmos-sdk/pubsub"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/bnb-chain/node/common/log"
	cmmtypes "github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/plugins/bridge/types"
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

	if token.GetContractAddress() == "" {
		return types.ErrTokenBound(fmt.Sprintf("token %s is not bound", msg.Symbol)).Result()
	}

	if !token.IsOwner(msg.From) {
		return sdk.ErrUnauthorized(fmt.Sprintf("only the owner can unbind token %s", msg.Symbol)).Result()
	}

	relayFee, sdkErr := types.GetFee(types.UnbindRelayFeeName)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	bscRelayFee, sdkErr := types.ConvertBCAmountToBSCAmount(types.BSCBNBDecimals, relayFee.Tokens.AmountOf(cmmtypes.NativeTokenSymbol))
	if sdkErr != nil {
		return sdkErr.Result()
	}

	_, sdkErr = keeper.BankKeeper.SendCoins(ctx, msg.From, types.PegAccount, relayFee.Tokens)
	if sdkErr != nil {
		log.With("module", "bridge").Error("send coins error", "err", sdkErr.Error())
		return sdkErr.Result()
	}

	unbindPackage := types.BindSynPackage{
		PackageType: types.BindTypeUnbind,
		TokenSymbol: types.SymbolToBytes(msg.Symbol),
	}

	encodedPackage, err := rlp.EncodeToBytes(unbindPackage)
	if err != nil {
		log.With("module", "bridge").Error("encode unbind package error", "err", err.Error())
		return sdk.ErrInternal("encode unbind package error").Result()
	}

	sendSeq, sdkErr := keeper.IbcKeeper.CreateRawIBCPackageByIdWithFee(ctx, keeper.DestChainId, types.BindChannelID, sdk.SynCrossChainPackageType,
		encodedPackage, *bscRelayFee.BigInt())
	if sdkErr != nil {
		log.With("module", "bridge").Error("create unbind ibc package error", "err", sdkErr.Error())
		return sdkErr.Result()
	}

	err = keeper.TokenMapper.UpdateBind(ctx, msg.Symbol, "", 0)
	if err != nil {
		log.With("module", "bridge").Error("update token info error", "err", err.Error())
		return sdk.ErrInternal(fmt.Sprintf("update token error, err=%s", err.Error())).Result()
	}

	log.With("module", "bridge").Info("unbind token success", "symbol", msg.Symbol, "contract_addr", token.GetContractDecimals())
	if ctx.IsDeliverTx() {
		keeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, msg.From})
		publishBindSuccessEvent(ctx, keeper, msg.From.String(), []pubsub.CrossReceiver{}, msg.Symbol, TransferUnBindType, relayFee.Tokens.AmountOf(cmmtypes.NativeTokenSymbol),
			token.GetContractAddress(), token.GetContractDecimals())
	}

	tags := sdk.NewTags(
		types.TagSendSequence, []byte(strconv.FormatUint(sendSeq, 10)),
		types.TagChannel, []byte{uint8(types.BindChannelID)},
		types.TagRelayerFee, []byte(strconv.FormatInt(relayFee.Tokens.AmountOf(cmmtypes.NativeTokenSymbol), 10)),
	)
	return sdk.Result{
		Tags: tags,
	}
}

func handleBindMsg(ctx sdk.Context, keeper Keeper, msg BindMsg) sdk.Result {
	if !time.Unix(msg.ExpireTime, 0).After(ctx.BlockHeader().Time.Add(types.MinBindExpireTimeGap)) {
		return types.ErrInvalidExpireTime(fmt.Sprintf("expire time should be %d seconds after now(%s)",
			int64(types.MinBindExpireTimeGap.Seconds()), ctx.BlockHeader().Time.UTC().String())).Result()
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

	bscTotalSupply, sdkErr := types.ConvertBCAmountToBSCAmount(msg.ContractDecimals, token.GetTotalSupply().ToInt64())
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

	bindPackage := types.BindSynPackage{
		PackageType:  types.BindTypeBind,
		TokenSymbol:  types.SymbolToBytes(msg.Symbol),
		ContractAddr: msg.ContractAddress,
		TotalSupply:  bscTotalSupply.BigInt(),
		PeggyAmount:  bscAmount.BigInt(),
		Decimals:     uint8(msg.ContractDecimals),
		ExpireTime:   uint64(msg.ExpireTime),
	}

	encodedPackage, err := rlp.EncodeToBytes(bindPackage)
	if err != nil {
		log.With("module", "bridge").Error("encode unbind package error", "err", err.Error())
		return sdk.ErrInternal("encode unbind package error").Result()
	}

	sendSeq, sdkErr := keeper.IbcKeeper.CreateRawIBCPackageByIdWithFee(ctx, keeper.DestChainId, types.BindChannelID, sdk.SynCrossChainPackageType, encodedPackage,
		*bscRelayFee.BigInt())
	if sdkErr != nil {
		log.With("module", "bridge").Error("create bind ibc package error", "err", sdkErr.Error())
		return sdkErr.Result()
	}

	if ctx.IsDeliverTx() {
		keeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, msg.From})
		publishCrossChainEvent(
			ctx,
			keeper,
			msg.From.String(),
			[]pubsub.CrossReceiver{
				{Addr: types.PegAccount.String(), Amount: bindRequest.DeductedAmount}},
			symbol,
			TransferBindType,
			relayFee.Tokens.AmountOf(cmmtypes.NativeTokenSymbol),
		)
	}
	pegTags := sdk.Tags{}
	for _, coin := range transferAmount {
		if coin.Amount > 0 {
			pegTags = append(pegTags, sdk.GetPegInTag(coin.Denom, coin.Amount))
		}
	}
	pegTags = append(pegTags, sdk.MakeTag(types.TagSendSequence, []byte(strconv.FormatUint(sendSeq, 10))))
	pegTags = append(pegTags, sdk.MakeTag(types.TagChannel, []byte{uint8(types.BindChannelID)}))
	pegTags = append(pegTags, sdk.MakeTag(types.TagRelayerFee, []byte(strconv.FormatInt(relayFee.Tokens.AmountOf(cmmtypes.NativeTokenSymbol), 10))))
	return sdk.Result{
		Tags: pegTags,
	}
}

func handleTransferOutMsg(ctx sdk.Context, keeper Keeper, msg TransferOutMsg) sdk.Result {
	if !time.Unix(msg.ExpireTime, 0).After(ctx.BlockHeader().Time.Add(types.MinTransferOutExpireTimeGap)) {
		return types.ErrInvalidExpireTime(fmt.Sprintf("expire time should be %d seconds after now(%s)",
			int64(types.MinTransferOutExpireTimeGap.Seconds()), ctx.BlockHeader().Time.UTC().String())).Result()
	}

	symbol := msg.Amount.Denom
	token, err := keeper.TokenMapper.GetToken(ctx, symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(fmt.Sprintf("symbol(%s) does not exist", symbol)).Result()
	}

	if token.GetContractAddress() == "" {
		return types.ErrTokenNotBound(fmt.Sprintf("token %s is not bound", symbol)).Result()
	}

	// check mini token
	sdkErr := bank.CheckAndValidateMiniTokenCoins(ctx, keeper.AccountKeeper, msg.From, sdk.Coins{msg.Amount})
	if sdkErr != nil {
		return sdkErr.Result()
	}

	relayFee, sdkErr := types.GetFee(types.TransferOutRelayFeeName)
	if sdkErr != nil {
		log.With("module", "bridge").Error("get transfer out syn fee error", "err", sdkErr.Error())
		return sdkErr.Result()
	}
	transferAmount := sdk.Coins{msg.Amount}.Plus(relayFee.Tokens)

	bscTransferAmount, sdkErr := types.ConvertBCAmountToBSCAmount(token.GetContractDecimals(), msg.Amount.Amount)
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

	contractAddr, err := sdk.NewSmartChainAddress(token.GetContractAddress())
	if err != nil {
		return types.ErrInvalidContractAddress(fmt.Sprintf("contract address is invalid, addr=%s", contractAddr)).Result()
	}
	transferPackage := types.TransferOutSynPackage{
		TokenSymbol:     types.SymbolToBytes(symbol),
		ContractAddress: contractAddr,
		RefundAddress:   msg.From.Bytes(),
		Recipient:       msg.To,
		Amount:          bscTransferAmount.BigInt(),
		ExpireTime:      uint64(msg.ExpireTime),
	}

	encodedPackage, err := rlp.EncodeToBytes(transferPackage)
	if err != nil {
		log.With("module", "bridge").Error("encode transfer out package error", "err", err.Error())
		return sdk.ErrInternal("encode unbind package error").Result()
	}

	sendSeq, sdkErr := keeper.IbcKeeper.CreateRawIBCPackageByIdWithFee(ctx, keeper.DestChainId, types.TransferOutChannelID, sdk.SynCrossChainPackageType,
		encodedPackage, *bscRelayFee.BigInt())
	if sdkErr != nil {
		log.With("module", "bridge").Error("create transfer out ibc package error", "err", sdkErr.Error())
		return sdkErr.Result()
	}

	if ctx.IsDeliverTx() {
		keeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, msg.From})
		publishCrossChainEvent(
			ctx,
			keeper,
			msg.From.String(),
			[]pubsub.CrossReceiver{
				{Addr: types.PegAccount.String(), Amount: msg.Amount.Amount}},
			symbol,
			TransferOutType,
			relayFee.Tokens.AmountOf(cmmtypes.NativeTokenSymbol),
		)
	}

	pegTags := sdk.Tags{}
	for _, coin := range transferAmount {
		if coin.Amount > 0 {
			pegTags = append(pegTags, sdk.GetPegInTag(coin.Denom, coin.Amount))
		}
	}
	pegTags = append(pegTags, sdk.MakeTag(types.TagSendSequence, []byte(strconv.FormatUint(sendSeq, 10))))
	pegTags = append(pegTags, sdk.MakeTag(types.TagChannel, []byte{uint8(types.TransferOutChannelID)}))
	pegTags = append(pegTags, sdk.MakeTag(types.TagRelayerFee, []byte(strconv.FormatInt(relayFee.Tokens.AmountOf(cmmtypes.NativeTokenSymbol), 10))))
	return sdk.Result{
		Tags: pegTags,
	}
}
