package bridge

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/bsc/rlp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"

	"github.com/binance-chain/node/common/log"
	ctypes "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
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
			publishBindSuccessEvent(ctx, app.bridgeKeeper, sdk.PegAccount.String(), []CrossReceiver{}, symbol, TransferApproveBindType, relayerFee, bindRequest.ContractAddress.String(), bindRequest.ContractDecimals)
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
			publishBindSuccessEvent(ctx, app.bridgeKeeper, types.PegAccount.String(), []CrossReceiver{
				{bindRequest.From.String(), bindRequest.DeductedAmount}}, symbol, TransferFailBindType, relayerFee, bindRequest.ContractAddress.String(), bindRequest.ContractDecimals)
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
		// allow transfer 0 amount tokens
		if sdk.IsUpgrade(upgrade.BEP100) {
			if amount.Int64() < 0 {
				return types.ErrInvalidAmount("amount to send should not be negative")
			}
		} else {
			if amount.Int64() <= 0 {
				return types.ErrInvalidAmount("amount to send should be positive")
			}
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
			Err:     sdk.ErrInsufficientFunds("balance of peg account is insufficient"),
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

var _ sdk.CrossChainApplication = &MirrorApp{}

type MirrorApp struct {
	bridgeKeeper Keeper
}

func NewMirrorApp(bridgeKeeper Keeper) *MirrorApp {
	return &MirrorApp{
		bridgeKeeper: bridgeKeeper,
	}
}

func (app *MirrorApp) ExecuteAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	log.With("module", "bridge").Error("received mirror ack package ")
	return sdk.ExecuteResult{}
}

func (app *MirrorApp) ExecuteFailAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	log.With("module", "bridge").Error("received mirror fail ack package ")
	return sdk.ExecuteResult{}
}

func (app *MirrorApp) checkMirrorSynPackage(ctx sdk.Context, mirrorPackage *types.MirrorSynPackage) uint8 {
	// check expire time
	if ctx.BlockHeader().Time.Unix() > int64(mirrorPackage.ExpireTime) {
		return types.MirrorErrCodeExpired
	}

	// check symbol
	symbol := types.BytesToSymbol(mirrorPackage.BEP20Symbol)
	err := ctypes.ValidateIssueSymbol(symbol)
	if err != nil {
		return types.MirrorErrCodeUnknown
	}

	// check supply
	supplyBigInt, cerr := types.ConvertBSCAmountToBCAmountBigInt(int8(mirrorPackage.BEP20Decimals), sdk.NewIntFromBigInt(mirrorPackage.BEP20TotalSupply))
	if cerr != nil {
		return types.MirrorErrCodeUnknown
	}
	maxSupply := sdk.NewInt(sdk.TokenMaxTotalSupply)
	if supplyBigInt.GT(maxSupply) {
		return types.MirrorErrCodeUnknown
	}

	return 0
}

func (app *MirrorApp) ExecuteSynPackage(ctx sdk.Context, payload []byte, relayerFee int64) sdk.ExecuteResult {
	mirrorPackage, sdkErr := types.DeserializeMirrorSynPackage(payload)
	if sdkErr != nil {
		log.With("module", "bridge").Error("unmarshal mirror claim error", "err", sdkErr.Error(), "claim", string(payload))
		panic("unmarshal mirror claim error")
	}

	errCode := app.checkMirrorSynPackage(ctx, mirrorPackage)
	if errCode != 0 {
		ackPackage, sdkErr := app.generateAckPackage(errCode, "", mirrorPackage)
		if sdkErr != nil {
			panic("generate ack package error")
		}

		return sdk.ExecuteResult{
			Payload: ackPackage,
		}
	}

	// check symbol existence
	symbol := app.getSymbol(payload, mirrorPackage)
	if exists := app.bridgeKeeper.TokenMapper.ExistsBEP2(ctx, symbol); exists {
		log.With("module", "bridge").Error("symbol already exists", "symbol", symbol)

		ackPackage, sdkErr := app.generateAckPackage(types.MirrorErrCodeBEP2SymbolExists, symbol, mirrorPackage)
		if sdkErr != nil {
			panic("generate ack package error")
		}

		return sdk.ExecuteResult{
			Payload: ackPackage,
		}
	}

	name := types.BytesToSymbol(mirrorPackage.BEP20Name)
	supply, sdkErr := types.ConvertBSCAmountToBCAmount(int8(mirrorPackage.BEP20Decimals), sdk.NewIntFromBigInt(mirrorPackage.BEP20TotalSupply))
	if sdkErr != nil {
		panic("convert bsc total supply error")
	}

	token, err := ctypes.NewToken(name, symbol, supply, types.PegAccount, true)
	if err != nil {
		panic(err.Error())
	}
	// set bep20 related fields
	token.SetContractAddress(mirrorPackage.ContractAddr.String())
	token.SetContractDecimals(int8(mirrorPackage.BEP20Decimals))

	// issue token and mint
	if err := app.bridgeKeeper.TokenMapper.NewToken(ctx, token); err != nil {
		panic(err.Error())
	}
	if _, _, sdkError := app.bridgeKeeper.BankKeeper.AddCoins(ctx, token.GetOwner(),
		sdk.Coins{{
			Denom:  token.GetSymbol(),
			Amount: token.GetTotalSupply().ToInt64(),
		}}); sdkError != nil {
		panic(sdkError.Error())
	}

	// return success payload
	ackPackage, sdkErr := app.generateAckPackage(0, symbol, mirrorPackage)
	if sdkErr != nil {
		panic("generate ack package error")
	}

	// add balance change accounts
	if ctx.IsDeliverTx() {
		addressesChanged := []sdk.AccAddress{types.PegAccount}
		app.bridgeKeeper.Pool.AddAddrs(addressesChanged)
	}

	// TODO: distribute fee

	return sdk.ExecuteResult{
		Payload: ackPackage,
	}
}

func (app *MirrorApp) generateAckPackage(code uint8, symbol string, synPackage *types.MirrorSynPackage) ([]byte, sdk.Error) {
	ackPackage := &types.MirrorAckPackage{
		MirrorSender: synPackage.MirrorSender,
		ContractAddr: synPackage.ContractAddr,
		Decimals:     synPackage.BEP20Decimals,
		BEP2Symbol:   types.SymbolToBytes(symbol),
		MirrorFee:    synPackage.MirrorFee,
		ErrorCode:    code,
	}

	encodedBytes, err := rlp.EncodeToBytes(ackPackage)
	if err != nil {
		return nil, sdk.ErrInternal("encode refund package error")
	}
	return encodedBytes, nil
}

func (app *MirrorApp) getSymbol(payload []byte, mirrorPackage *types.MirrorSynPackage) string {
	symbol := types.BytesToSymbol(mirrorPackage.BEP20Symbol)
	symbol = strings.ToUpper(symbol)

	suffix := app.getBep2TokenSuffix(payload)

	symbol = fmt.Sprintf("%s-%s", symbol, suffix)
	return symbol
}

func (app *MirrorApp) getBep2TokenSuffix(payload []byte) string {
	payloadHash := cmn.HexBytes(tmhash.Sum(payload)).String()

	suffix := payloadHash[:ctypes.TokenSymbolTxHashSuffixLen]
	return suffix
}

var _ sdk.CrossChainApplication = &MirrorSyncApp{}

type MirrorSyncApp struct {
	bridgeKeeper Keeper
}

func NewMirrorSyncApp(bridgeKeeper Keeper) *MirrorSyncApp {
	return &MirrorSyncApp{
		bridgeKeeper: bridgeKeeper,
	}
}

func (app *MirrorSyncApp) ExecuteAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	log.With("module", "bridge").Error("received mirror sync ack package ")
	return sdk.ExecuteResult{}
}

func (app *MirrorSyncApp) ExecuteFailAckPackage(ctx sdk.Context, payload []byte) sdk.ExecuteResult {
	log.With("module", "bridge").Error("received mirror sync fail ack package ")
	return sdk.ExecuteResult{}
}
func (app *MirrorSyncApp) checkMirrorSyncPackage(ctx sdk.Context, mirrorSyncPackage *types.MirrorSyncSynPackage) uint8 {
	// check expire time
	if ctx.BlockHeader().Time.Unix() > int64(mirrorSyncPackage.ExpireTime) {
		return types.MirrorSyncErrCodeExpired
	}

	return 0
}

func (app *MirrorSyncApp) ExecuteSynPackage(ctx sdk.Context, payload []byte, relayerFee int64) sdk.ExecuteResult {
	mirrorSyncPackage, sdkErr := types.DeserializeMirrorSyncSynPackage(payload)
	if sdkErr != nil {
		log.With("module", "bridge").Error("unmarshal mirror sync claim error", "err", sdkErr.Error(), "claim", string(payload))
		panic("unmarshal mirror claim error")
	}

	errCode := app.checkMirrorSyncPackage(ctx, mirrorSyncPackage)
	if errCode != 0 {
		ackPackage, sdkErr := app.generateAckPackage(errCode, mirrorSyncPackage)
		if sdkErr != nil {
			panic("generate ack package error")
		}

		return sdk.ExecuteResult{
			Payload: ackPackage,
		}
	}

	symbol := types.BytesToSymbol(mirrorSyncPackage.BEP2Symbol)
	token, err := app.bridgeKeeper.TokenMapper.GetToken(ctx, symbol)
	if err != nil {
		panic("get bep 2 token error")
	}

	// check token
	if token.GetContractAddress() == "" || token.GetOwner().String() != types.PegAccount.String() {
		ackPackage, sdkErr := app.generateAckPackage(types.MirrorSyncErrNotBoundByMirror, mirrorSyncPackage)
		if sdkErr != nil {
			panic("generate ack package error")
		}

		return sdk.ExecuteResult{
			Payload: ackPackage,
		}
	}

	// mint or burn
	newSupply := mirrorSyncPackage.BEP20TotalSupply.Int64()
	if newSupply > ctypes.TokenMaxTotalSupply {
		ackPackage, sdkErr := app.generateAckPackage(types.MirrorSyncErrCodeUnknown, mirrorSyncPackage)
		if sdkErr != nil {
			panic("generate ack package error")
		}

		return sdk.ExecuteResult{
			Payload: ackPackage,
		}
	}

	if newSupply > token.GetTotalSupply().ToInt64() {
		if _, _, sdkError := app.bridgeKeeper.BankKeeper.AddCoins(ctx, token.GetOwner(),
			sdk.Coins{{
				Denom:  token.GetSymbol(),
				Amount: newSupply - token.GetTotalSupply().ToInt64(),
			}}); sdkError != nil {
			panic(sdkError.Error())
		}
	} else if newSupply < token.GetTotalSupply().ToInt64() {
		if _, _, sdkError := app.bridgeKeeper.BankKeeper.SubtractCoins(ctx, token.GetOwner(),
			sdk.Coins{{
				Denom:  token.GetSymbol(),
				Amount: token.GetTotalSupply().ToInt64() - newSupply,
			}}); sdkError != nil {
			panic(sdkError.Error())
		}
	}

	// add balance change accounts
	if newSupply != token.GetTotalSupply().ToInt64() && ctx.IsDeliverTx() {
		addressesChanged := []sdk.AccAddress{types.PegAccount}
		app.bridgeKeeper.Pool.AddAddrs(addressesChanged)
	}

	// TODO: distribute fee

	return sdk.ExecuteResult{}
}

func (app *MirrorSyncApp) generateAckPackage(code uint8, synPackage *types.MirrorSyncSynPackage) ([]byte, sdk.Error) {
	ackPackage := &types.MirrorSyncAckPackage{
		SyncSender:   synPackage.SyncSender,
		ContractAddr: synPackage.ContractAddr,
		SyncFee:      synPackage.SyncFee,
		ErrorCode:    code,
	}

	encodedBytes, err := rlp.EncodeToBytes(ackPackage)
	if err != nil {
		return nil, sdk.ErrInternal("encode refund package error")
	}
	return encodedBytes, nil
}
