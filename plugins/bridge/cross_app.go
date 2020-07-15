package bridge

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/plugins/bridge/types"
)

var _ sdk.CrossChainApplication = &BindApp{}

type BindApp struct {
	bridgeKeeper Keeper
}

func NewBindApp(bridgeKeeper Keeper) *BindApp {
	return &BindApp{
		bridgeKeeper: bridgeKeeper,
	}
}

func (app *BindApp) ExecuteAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	log.With("module", "bridge").Info("received bind ack package")
	return sdk.ExecuteResult{}
}

func (app *BindApp) ExecuteFailAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	log.With("module", "bridge").Info("received bind fail ack package")
	bindPackage, sdkErr := types.DeserializeBindSynPackage(payload)
	if sdkErr != nil {
		return sdk.ExecuteResult{
			Err: sdkErr,
		}
	}

	symbol := types.BytesToSymbol(bindPackage.TokenSymbol)
	bindRequest, sdkErr := app.bridgeKeeper.GetBindRequest(ctx, symbol)
	if sdkErr != nil {
		return sdk.ExecuteResult{
			Err: sdkErr,
		}
	}

	_, sdkErr = app.bridgeKeeper.BankKeeper.SendCoins(ctx, types.PegAccount, bindRequest.From,
		sdk.Coins{sdk.Coin{Denom: bindRequest.Symbol, Amount: bindRequest.DeductedAmount}})
	if sdkErr != nil {
		log.With("module", "bridge").Error("send coins error", "err", sdkErr.Error())
		return sdk.ExecuteResult{
			Err: sdkErr,
		}
	}

	app.bridgeKeeper.DeleteBindRequest(ctx, symbol)

	if ctx.IsDeliverTx() {
		app.bridgeKeeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, bindRequest.From})
		publishCrossChainEvent(ctx, app.bridgeKeeper, types.PegAccount.String(), []CrossReceiver{
			{bindRequest.From.String(), bindRequest.DeductedAmount}}, symbol, TransferFailBindType, 0)
	}
	return sdk.ExecuteResult{}
}

func (app *BindApp) ExecuteSynPackage(ctx sdk.Context, payload []byte, relayerFee int64) sdk.ExecuteResult {
	approvePackage, sdkErr := types.DeserializeApproveBindSynPackage(payload)
	if sdkErr != nil {
		return sdk.ExecuteResult{
			Err: sdkErr,
		}
	}

	symbol := types.BytesToSymbol(approvePackage.TokenSymbol)

	bindRequest, sdkErr := app.bridgeKeeper.GetBindRequest(ctx, symbol)
	if sdkErr != nil {
		return sdk.ExecuteResult{
			Err: sdkErr,
		}
	}

	if bindRequest.Symbol != symbol {
		return sdk.ExecuteResult{
			Err: types.ErrInvalidClaim(fmt.Sprintf("approve symbol(%s) is not identical to bind request symbol(%s)", symbol, bindRequest.Symbol)),
		}
	}

	log.With("module", "bridge").Info("update bind status", "status", approvePackage.Status.String(), "symbol", symbol)
	if approvePackage.Status == types.BindStatusSuccess {
		sdkErr := app.bridgeKeeper.TokenMapper.UpdateBind(ctx, bindRequest.Symbol,
			bindRequest.ContractAddress.String(), bindRequest.ContractDecimals)

		if sdkErr != nil {
			log.With("module", "bridge").Error("update token info error", "err", sdkErr.Error(), "symbol", symbol)
			return sdk.ExecuteResult{
				Err: sdk.ErrInternal(fmt.Sprintf("update token bind info error")),
			}
		}

		app.bridgeKeeper.SetContractDecimals(ctx, bindRequest.ContractAddress, bindRequest.ContractDecimals)
		if ctx.IsDeliverTx() {
			publishCrossChainEvent(ctx, app.bridgeKeeper, bindRequest.From.String(), []CrossReceiver{}, symbol, TransferApproveBindType, relayerFee)
		}
		log.With("module", "bridge").Info("bind token success", "symbol", symbol, "contract_addr", bindRequest.ContractAddress.String())
	} else {
		_, sdkErr = app.bridgeKeeper.BankKeeper.SendCoins(ctx, types.PegAccount, bindRequest.From,
			sdk.Coins{sdk.Coin{Denom: bindRequest.Symbol, Amount: bindRequest.DeductedAmount}})
		if sdkErr != nil {
			log.With("module", "bridge").Error("send coins error", "err", sdkErr.Error())
			return sdk.ExecuteResult{
				Err: sdkErr,
			}
		}

		if ctx.IsDeliverTx() {
			app.bridgeKeeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, bindRequest.From})
			publishCrossChainEvent(ctx, app.bridgeKeeper, types.PegAccount.String(), []CrossReceiver{
				{bindRequest.From.String(), bindRequest.DeductedAmount}}, symbol, TransferFailBindType, relayerFee)
		}
	}

	app.bridgeKeeper.DeleteBindRequest(ctx, symbol)
	return sdk.ExecuteResult{}
}

var _ sdk.CrossChainApplication = &TransferOutApp{}

type TransferOutApp struct {
	bridgeKeeper Keeper
}

func NewTransferOutApp(bridgeKeeper Keeper) *TransferOutApp {
	return &TransferOutApp{
		bridgeKeeper: bridgeKeeper,
	}
}

func (app *TransferOutApp) checkPackage(refundPackage *types.TransferOutRefundPackage) sdk.Error {
	if len(refundPackage.RefundAddr) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(refundPackage.RefundAddr.String())
	}

	if refundPackage.RefundAmount.Int64() < 0 {
		return types.ErrInvalidAmount("amount to refund should be positive")
	}
	return nil
}

func (app *TransferOutApp) ExecuteAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	if len(payload) == 0 {
		log.With("module", "bridge").Info("receive transfer out ack package")
		return sdk.ExecuteResult{}
	}

	log.With("module", "bridge").Info("receive transfer out refund ack package")

	refundPackage, sdkErr := types.DeserializeTransferOutRefundPackage(payload)
	if sdkErr != nil {
		log.With("module", "bridge").Error("unmarshal transfer out refund claim error", "err", sdkErr.Error(), "claim", string(payload))
		return sdk.ExecuteResult{
			Err: sdkErr,
		}
	}

	sdkErr = app.checkPackage(refundPackage)
	if sdkErr != nil {
		log.With("module", "bridge").Error("check transfer out refund package error", "err", sdkErr.Error(), "claim", string(payload))
		return sdk.ExecuteResult{
			Err: sdkErr,
		}
	}

	symbol := types.BytesToSymbol(refundPackage.TokenSymbol)
	_, sdkErr = app.bridgeKeeper.BankKeeper.SendCoins(ctx, types.PegAccount, refundPackage.RefundAddr,
		sdk.Coins{
			sdk.Coin{
				Denom:  symbol,
				Amount: refundPackage.RefundAmount.Int64(),
			},
		},
	)
	if sdkErr != nil {
		log.With("module", "bridge").Error("send coins error", "err", sdkErr.Error())
		return sdk.ExecuteResult{
			Err: sdkErr,
		}
	}

	if ctx.IsDeliverTx() {
		app.bridgeKeeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, refundPackage.RefundAddr})
		publishCrossChainEvent(ctx, app.bridgeKeeper, types.PegAccount.String(), []CrossReceiver{
			{refundPackage.RefundAddr.String(), refundPackage.RefundAmount.Int64()}}, symbol, TransferAckRefundType, 0)
	}
	return sdk.ExecuteResult{
		Tags: sdk.Tags{sdk.GetPegOutTag(symbol, refundPackage.RefundAmount.Int64())},
	}
}

func (app *TransferOutApp) ExecuteFailAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	log.With("module", "bridge").Info("received transfer out fail ack package")

	transferOutPackage, sdkErr := types.DeserializeTransferOutSynPackage(payload)
	if sdkErr != nil {
		return sdk.ExecuteResult{
			Err: sdkErr,
		}
	}

	contractDecimals := app.bridgeKeeper.GetContractDecimals(ctx, transferOutPackage.ContractAddress)
	bcAmount, sdkErr := types.ConvertBSCAmountToBCAmount(contractDecimals, sdk.NewIntFromBigInt(transferOutPackage.Amount))
	if sdkErr != nil {
		return sdk.ExecuteResult{
			Err: sdkErr,
		}
	}

	symbol := types.BytesToSymbol(transferOutPackage.TokenSymbol)
	_, sdkErr = app.bridgeKeeper.BankKeeper.SendCoins(ctx, types.PegAccount, transferOutPackage.RefundAddress,
		sdk.Coins{
			sdk.Coin{
				Denom:  symbol,
				Amount: bcAmount,
			},
		},
	)

	if sdkErr != nil {
		log.With("module", "bridge").Error("send coins error", "err", sdkErr.Error())
		return sdk.ExecuteResult{
			Err: sdkErr,
		}
	}

	if ctx.IsDeliverTx() {
		app.bridgeKeeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, transferOutPackage.RefundAddress})
		publishCrossChainEvent(ctx, app.bridgeKeeper, types.PegAccount.String(), []CrossReceiver{
			{transferOutPackage.RefundAddress.String(), bcAmount}}, symbol, TransferFailAckRefundType, 0)
	}

	return sdk.ExecuteResult{
		Tags: sdk.Tags{sdk.GetPegOutTag(symbol, bcAmount)},
	}
}

func (app *TransferOutApp) ExecuteSynPackage(ctx sdk.Context, payload []byte, _ int64) sdk.ExecuteResult {
	log.With("module", "bridge").Error("received transfer out syn package ")
	return sdk.ExecuteResult{}
}

var _ sdk.CrossChainApplication = &TransferInApp{}

type TransferInApp struct {
	bridgeKeeper Keeper
}

func NewTransferInApp(bridgeKeeper Keeper) *TransferInApp {
	return &TransferInApp{
		bridgeKeeper: bridgeKeeper,
	}
}

func (app *TransferInApp) checkTransferInSynPackage(transferInPackage *types.TransferInSynPackage) sdk.Error {
	if len(transferInPackage.Amounts) == 0 {
		return types.ErrInvalidLength("length of Amounts should not be 0")
	}

	if len(transferInPackage.RefundAddresses) != len(transferInPackage.ReceiverAddresses) ||
		len(transferInPackage.RefundAddresses) != len(transferInPackage.Amounts) {
		return types.ErrInvalidLength("length of RefundAddresses, ReceiverAddresses, Amounts should be the same")
	}

	for _, addr := range transferInPackage.RefundAddresses {
		if addr.IsEmpty() {
			return types.ErrInvalidEthereumAddress("refund address should not be empty")
		}
	}

	for _, addr := range transferInPackage.ReceiverAddresses {
		if len(addr) != sdk.AddrLen {
			return sdk.ErrInvalidAddress(fmt.Sprintf("length of receiver addreess should be %d", sdk.AddrLen))
		}
	}

	for _, amount := range transferInPackage.Amounts {
		if amount.Int64() <= 0 {
			return types.ErrInvalidAmount("amount to send should be positive")
		}
	}

	return nil
}

func (app *TransferInApp) ExecuteAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	log.With("module", "bridge").Error("received transfer in ack package ")
	return sdk.ExecuteResult{}
}

func (app *TransferInApp) ExecuteFailAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	log.With("module", "bridge").Error("received transfer in fail ack package ")
	return sdk.ExecuteResult{}
}

func (app *TransferInApp) ExecuteSynPackage(ctx sdk.Context, payload []byte, relayerFee int64) sdk.ExecuteResult {
	transferInPackage, sdkErr := types.DeserializeTransferInSynPackage(payload)
	if sdkErr != nil {
		log.With("module", "bridge").Error("unmarshal transfer in claim error", "err", sdkErr.Error(), "claim", string(payload))
		panic("unmarshal transfer in claim error")
	}

	sdkErr = app.checkTransferInSynPackage(transferInPackage)
	if sdkErr != nil {
		log.With("module", "bridge").Error("check transfer in package error", "err", sdkErr.Error(), "claim", string(payload))
		panic(sdkErr)
	}

	symbol := types.BytesToSymbol(transferInPackage.TokenSymbol)
	tokenInfo, err := app.bridgeKeeper.TokenMapper.GetToken(ctx, symbol)
	if err != nil {
		panic(err)
	}

	if tokenInfo.GetContractAddress() != transferInPackage.ContractAddress.String() {
		// check decimals of contract
		contractDecimals := app.bridgeKeeper.GetContractDecimals(ctx, transferInPackage.ContractAddress)
		if contractDecimals < 0 {
			log.With("module", "bridge").Error("decimals of contract does not exist", "contract_addr", transferInPackage.ContractAddress.String())
			panic(fmt.Sprintf("decimals of contract does not exist, contract_addr=%s",
				transferInPackage.ContractAddress.String()))
		}

		refundPackage, sdkErr := app.bridgeKeeper.RefundTransferIn(contractDecimals, transferInPackage, types.UnboundToken)
		if sdkErr != nil {
			log.With("module", "bridge").Error("refund transfer in error", "err", sdkErr.Error())
			panic(sdkErr)
		}
		return sdk.ExecuteResult{
			Payload: refundPackage,
			Err:     types.ErrTokenBindRelationChanged("contract addr mismatch"),
		}
	}

	if int64(transferInPackage.ExpireTime) < ctx.BlockHeader().Time.Unix() {
		refundPackage, sdkErr := app.bridgeKeeper.RefundTransferIn(tokenInfo.GetContractDecimals(), transferInPackage, types.Timeout)
		if sdkErr != nil {
			log.With("module", "bridge").Error("refund transfer in error", "err", sdkErr.Error())
			panic(sdkErr)
		}
		return sdk.ExecuteResult{
			Payload: refundPackage,
			Err:     types.ErrTransferInExpire("the package is expired"),
		}
	}

	balance := app.bridgeKeeper.BankKeeper.GetCoins(ctx, types.PegAccount)
	var totalTransferInAmount sdk.Coins
	for idx := range transferInPackage.ReceiverAddresses {
		amount := sdk.NewCoin(symbol, transferInPackage.Amounts[idx].Int64())
		totalTransferInAmount = sdk.Coins{amount}.Plus(totalTransferInAmount)
	}

	if !balance.IsGTE(totalTransferInAmount) {
		refundPackage, sdkErr := app.bridgeKeeper.RefundTransferIn(tokenInfo.GetContractDecimals(), transferInPackage, types.InsufficientBalance)
		if sdkErr != nil {
			log.With("module", "bridge").Error("refund transfer in error", "err", sdkErr.Error())
			panic(sdkErr)
		}
		return sdk.ExecuteResult{
			Payload: refundPackage,
		}
	}

	for idx, receiverAddr := range transferInPackage.ReceiverAddresses {
		amount := sdk.NewCoin(symbol, transferInPackage.Amounts[idx].Int64())
		_, sdkErr = app.bridgeKeeper.BankKeeper.SendCoins(ctx, types.PegAccount, receiverAddr, sdk.Coins{amount})
		if sdkErr != nil {
			log.With("module", "bridge").Error("send coins error", "err", sdkErr.Error())
			panic(sdkErr)
		}
	}

	if ctx.IsDeliverTx() {
		addressesChanged := append(transferInPackage.ReceiverAddresses, types.PegAccount)
		app.bridgeKeeper.Pool.AddAddrs(addressesChanged)
		to := make([]CrossReceiver, 0, len(transferInPackage.ReceiverAddresses))
		for idx, receiverAddr := range transferInPackage.ReceiverAddresses {
			to = append(to, CrossReceiver{
				Addr:   receiverAddr.String(),
				Amount: transferInPackage.Amounts[idx].Int64(),
			})
		}
		publishCrossChainEvent(ctx, app.bridgeKeeper, types.PegAccount.String(), to, symbol, TransferInType, relayerFee)
	}

	// emit peg related event
	var totalAmount int64 = 0
	var tags sdk.Tags
	if totalTransferInAmount != nil {
		totalAmount = totalTransferInAmount.AmountOf(symbol)
		tags = sdk.Tags{sdk.GetPegOutTag(symbol, totalAmount)}
	}

	return sdk.ExecuteResult{
		Tags: tags,
	}
}
