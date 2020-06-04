package bridge

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/fees"

	"github.com/binance-chain/node/common/log"
	cmmtypes "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/bridge/types"
)

var _ sdk.ClaimHooks = &SkipSequenceClaimHooks{}

type SkipSequenceClaimHooks struct {
	bridgeKeeper Keeper
}

func NewSkipSequenceClaimHooks(bridgeKeeper Keeper) *SkipSequenceClaimHooks {
	return &SkipSequenceClaimHooks{
		bridgeKeeper: bridgeKeeper,
	}
}

func (hooks *SkipSequenceClaimHooks) CheckClaim(ctx sdk.Context, claim string) sdk.Error {
	skipSequenceClaim, err := types.GetSkipSequenceClaimFromOracleClaim(claim)
	if err != nil {
		return types.ErrInvalidClaim(fmt.Sprintf("unmarshal skip sequence claim error, claim=%s", claim))
	}

	if skipSequenceClaim.ClaimType == types.ClaimTypeSkipSequence {
		return types.ErrInvalidClaim(fmt.Sprintf("can not skip claim type %d", skipSequenceClaim.ClaimType))
	}

	claimTypeName := hooks.bridgeKeeper.OracleKeeper.GetClaimTypeName(skipSequenceClaim.ClaimType)
	if claimTypeName == "" {
		return types.ErrInvalidClaim(fmt.Sprintf("claim type %d does not exist", skipSequenceClaim.ClaimType))
	}

	if skipSequenceClaim.Sequence < 0 {
		return types.ErrInvalidSequence("sequence should be larger than 0")
	}

	currentSeq := hooks.bridgeKeeper.OracleKeeper.GetCurrentSequence(ctx, skipSequenceClaim.ClaimType)
	if skipSequenceClaim.Sequence != currentSeq {
		return types.ErrInvalidSequence(fmt.Sprintf("current sequence is %d", currentSeq))
	}
	return nil
}

func (hooks *SkipSequenceClaimHooks) ExecuteClaim(ctx sdk.Context, claim string) (sdk.ClaimResult, sdk.Error) {
	skipSequenceClaim, err := types.GetSkipSequenceClaimFromOracleClaim(claim)
	if err != nil {
		log.With("module", "bridge").Error("unmarshal skip sequence claim error", "err", err.Error(), "claim", claim)
		return sdk.ClaimResult{}, types.ErrInvalidClaim(fmt.Sprintf("unmarshal claim error, claim=%s", claim))
	}

	hooks.bridgeKeeper.OracleKeeper.IncreaseSequence(ctx, skipSequenceClaim.ClaimType)

	return sdk.ClaimResult{
		Code: 0,
		Msg:  "",
	}, nil
}

var _ sdk.ClaimHooks = &UpdateBindClaimHooks{}

type UpdateBindClaimHooks struct {
	bridgeKeeper Keeper
}

func NewUpdateBindClaimHooks(bridgeKeeper Keeper) *UpdateBindClaimHooks {
	return &UpdateBindClaimHooks{
		bridgeKeeper: bridgeKeeper,
	}
}

func (hooks *UpdateBindClaimHooks) CheckClaim(ctx sdk.Context, claim string) sdk.Error {
	updateBindClaim, err := types.GetUpdateBindClaimFromOracleClaim(claim)
	if err != nil {
		return types.ErrInvalidClaim(fmt.Sprintf("unmarshal update bind claim error, claim=%s", claim))
	}

	if len(updateBindClaim.Symbol) == 0 {
		return types.ErrInvalidSymbol("symbol should not be empty")
	}

	if updateBindClaim.Status != types.BindStatusSuccess &&
		updateBindClaim.Status != types.BindStatusRejected &&
		updateBindClaim.Status != types.BindStatusTimeout &&
		updateBindClaim.Status != types.BindStatusInvalidParameter {
		return types.ErrInvalidStatus(fmt.Sprintf("status(%d) does not exist", updateBindClaim.Status))
	}

	if updateBindClaim.ContractAddress.IsEmpty() {
		return types.ErrInvalidContractAddress("contract address should not be empty")
	}

	if updateBindClaim.Symbol == cmmtypes.NativeTokenSymbol {
		return types.ErrInvalidSymbol(fmt.Sprintf("can not bind native token(%s)", updateBindClaim.Symbol))
	}

	if _, err := hooks.bridgeKeeper.TokenMapper.GetToken(ctx, updateBindClaim.Symbol); err != nil {
		return types.ErrInvalidSymbol(fmt.Sprintf("token %s does not exist", updateBindClaim.Symbol))
	}

	return nil
}

func (hooks *UpdateBindClaimHooks) ExecuteClaim(ctx sdk.Context, claim string) (sdk.ClaimResult, sdk.Error) {
	updateBindClaim, sdkErr := types.GetUpdateBindClaimFromOracleClaim(claim)
	if sdkErr != nil {
		return sdk.ClaimResult{}, sdkErr
	}

	bindRequest, sdkErr := hooks.bridgeKeeper.GetBindRequest(ctx, updateBindClaim.Symbol)
	if sdkErr != nil {
		return sdk.ClaimResult{}, sdkErr
	}

	isIdentical := bindRequest.Symbol == updateBindClaim.Symbol &&
		bindRequest.ContractAddress.String() == updateBindClaim.ContractAddress.String()
	if !isIdentical {
		return sdk.ClaimResult{}, types.ErrInvalidClaim("claim is not identical to bind request")
	}

	log.With("module", "bridge").Info("update bind status", "status", updateBindClaim.Status.String(), "symbol", updateBindClaim.Symbol)
	if updateBindClaim.Status == types.BindStatusSuccess {
		sdkErr := hooks.bridgeKeeper.TokenMapper.UpdateBind(ctx, bindRequest.Symbol,
			bindRequest.ContractAddress.String(), bindRequest.ContractDecimals)

		if sdkErr != nil {
			log.With("module", "bridge").Error("update token info error", "err", sdkErr.Error(), "symbol", updateBindClaim.Symbol)
			return sdk.ClaimResult{}, sdk.ErrInternal(fmt.Sprintf("update token bind info error"))
		}

		hooks.bridgeKeeper.SetContractDecimals(ctx, bindRequest.ContractAddress, bindRequest.ContractDecimals)

		log.With("module", "bridge").Info("bind token success", "symbol", updateBindClaim.Symbol, "contract_addr", updateBindClaim.ContractAddress.String())
	} else {
		_, sdkErr = hooks.bridgeKeeper.BankKeeper.SendCoins(ctx, types.PegAccount, bindRequest.From,
			sdk.Coins{sdk.Coin{Denom: bindRequest.Symbol, Amount: bindRequest.DeductedAmount}})
		if sdkErr != nil {
			log.With("module", "bridge").Error("send coins error", "err", sdkErr.Error())
			return sdk.ClaimResult{}, sdkErr
		}

		if ctx.IsDeliverTx() {
			hooks.bridgeKeeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, bindRequest.From})
		}
	}

	hooks.bridgeKeeper.DeleteBindRequest(ctx, updateBindClaim.Symbol)
	return sdk.ClaimResult{
		Code: 0,
		Msg:  "",
	}, nil
}

var _ sdk.ClaimHooks = &TransferOutRefundClaimHooks{}

type TransferOutRefundClaimHooks struct {
	bridgeKeeper Keeper
}

func NewTransferOutRefundClaimHooks(bridgeKeeper Keeper) *TransferOutRefundClaimHooks {
	return &TransferOutRefundClaimHooks{
		bridgeKeeper: bridgeKeeper,
	}
}

func (hooks *TransferOutRefundClaimHooks) CheckClaim(ctx sdk.Context, claim string) sdk.Error {
	refundClaim, sdkErr := types.GetTransferOutRefundClaimFromOracleClaim(claim)
	if sdkErr != nil {
		return types.ErrInvalidClaim(fmt.Sprintf("unmarshal transfer out refund claim error, claim=%s", claim))
	}

	if refundClaim.TransferOutSequence < 0 {
		return types.ErrInvalidSequence("transfer out sequence should not be less than 0")
	}

	if len(refundClaim.RefundAddress) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(refundClaim.RefundAddress.String())
	}

	if !refundClaim.Amount.IsPositive() {
		return types.ErrInvalidAmount("amount to refund should be positive")
	}

	if refundClaim.RefundReason.String() == "" {
		return types.ErrInvalidStatus(fmt.Sprintf("refund reason(%d) does not exist", refundClaim.RefundReason))
	}

	ibcPackage, err := hooks.bridgeKeeper.IbcKeeper.GetIBCPackage(ctx, hooks.bridgeKeeper.DestChainId, types.TransferOutChannel, uint64(refundClaim.TransferOutSequence))
	if err != nil {
		return types.ErrInvalidClaim(fmt.Sprintf("transfer out ibc package does not exist, sequence=%d", refundClaim.TransferOutSequence))
	}

	transferOutPackage, err := types.DeserializeTransferOutPackage(ibcPackage)
	if err != nil {
		return types.ErrDeserializePackageFailed(fmt.Sprintf("deserialize transfer out package error"))
	}

	if bytes.Compare(transferOutPackage.RefundAddress, refundClaim.RefundAddress) != 0 ||
		transferOutPackage.Bep2TokenSymbol != refundClaim.Amount.Denom {
		return types.ErrInvalidClaim(fmt.Sprintf("transfer out symbol(%s) is not identical to refund symbol(%s)",
			transferOutPackage.Bep2TokenSymbol, refundClaim.Amount.Denom))
	}

	if bytes.Compare(transferOutPackage.RefundAddress, refundClaim.RefundAddress) != 0 {
		return types.ErrInvalidClaim(fmt.Sprintf("transfer out address(%s) is not identical to refund address(%s)",
			sdk.AccAddress(transferOutPackage.RefundAddress).String(), refundClaim.RefundAddress.String()))
	}

	contractAddr := types.SmartChainAddress{}
	contractAddr.SetBytes(transferOutPackage.ContractAddress)
	contractDecimals := hooks.bridgeKeeper.GetContractDecimals(ctx, contractAddr)

	bcAmount, sdkErr := types.ConvertBSCAmountToBCAmount(contractDecimals, transferOutPackage.Amount)
	if sdkErr != nil {
		return sdkErr
	}
	if bcAmount != refundClaim.Amount.Amount {
		return types.ErrInvalidClaim(fmt.Sprintf("transfer out amount(%d) is not equal to refund amount(%d)",
			bcAmount, refundClaim.Amount.Amount))
	}

	return nil
}

func (hooks *TransferOutRefundClaimHooks) ExecuteClaim(ctx sdk.Context, claim string) (sdk.ClaimResult, sdk.Error) {
	refundClaim, sdkErr := types.GetTransferOutRefundClaimFromOracleClaim(claim)
	if sdkErr != nil {
		log.With("module", "bridge").Error("unmarshal transfer out refund claim error", "err", sdkErr.Error(), "claim", claim)
		return sdk.ClaimResult{}, sdkErr
	}

	_, sdkErr = hooks.bridgeKeeper.BankKeeper.SendCoins(ctx, types.PegAccount, refundClaim.RefundAddress, sdk.Coins{refundClaim.Amount})
	if sdkErr != nil {
		log.With("module", "bridge").Error("send coins error", "err", sdkErr.Error())
		return sdk.ClaimResult{}, sdkErr
	}

	if ctx.IsDeliverTx() {
		hooks.bridgeKeeper.Pool.AddAddrs([]sdk.AccAddress{types.PegAccount, refundClaim.RefundAddress})
	}

	return sdk.ClaimResult{
		Code: 0,
		Msg:  "",
		Tags: nil,
	}, nil
}

var _ sdk.ClaimHooks = &TransferInClaimHooks{}

type TransferInClaimHooks struct {
	bridgeKeeper Keeper
}

func NewTransferInClaimHooks(bridgeKeeper Keeper) *TransferInClaimHooks {
	return &TransferInClaimHooks{
		bridgeKeeper: bridgeKeeper,
	}
}

func (hooks *TransferInClaimHooks) CheckClaim(ctx sdk.Context, claim string) sdk.Error {
	transferInClaim, err := types.GetTransferInClaimFromOracleClaim(claim)
	if err != nil {
		return types.ErrInvalidClaim(fmt.Sprintf("unmarshal transfer in claim error, claim=%s", claim))
	}

	if transferInClaim.ExpireTime <= 0 {
		return types.ErrInvalidExpireTime("expire time should be larger than 0")
	}

	if len(transferInClaim.Symbol) == 0 {
		return types.ErrInvalidSymbol("length of symbol should not be 0")
	}

	if transferInClaim.Symbol != cmmtypes.NativeTokenSymbol && transferInClaim.ContractAddress.IsEmpty() {
		return types.ErrInvalidEthereumAddress("contract address should not be empty")
	}

	if len(transferInClaim.RefundAddresses) == 0 {
		return types.ErrInvalidLength("length of RefundAddresses should not be 0")
	}

	for _, addr := range transferInClaim.RefundAddresses {
		if addr.IsEmpty() {
			return types.ErrInvalidEthereumAddress("refund address should not be empty")
		}
	}

	if len(transferInClaim.ReceiverAddresses) == 0 {
		return types.ErrInvalidLength("length of ReceiverAddresses should not be 0")
	}

	for _, addr := range transferInClaim.ReceiverAddresses {
		if len(addr) != sdk.AddrLen {
			return sdk.ErrInvalidAddress(fmt.Sprintf("length of receiver addreess should be %d", sdk.AddrLen))
		}
	}

	if len(transferInClaim.Amounts) == 0 {
		return types.ErrInvalidLength("length of Amounts should not be 0")
	}

	for _, amount := range transferInClaim.Amounts {
		if amount <= 0 {
			return types.ErrInvalidAmount("amount to send should be positive")
		}
	}

	if len(transferInClaim.RefundAddresses) != len(transferInClaim.ReceiverAddresses) ||
		len(transferInClaim.RefundAddresses) != len(transferInClaim.Amounts) {
		return types.ErrInvalidLength("length of RefundAddresses, ReceiverAddresses, Amounts should be the same")
	}

	if !transferInClaim.RelayFee.IsPositive() {
		return types.ErrInvalidAmount("relay fee amount should be positive")
	}

	return nil
}

func (hooks *TransferInClaimHooks) ExecuteClaim(ctx sdk.Context, claim string) (sdk.ClaimResult, sdk.Error) {
	transferInClaim, sdkErr := types.GetTransferInClaimFromOracleClaim(claim)
	if sdkErr != nil {
		log.With("module", "bridge").Error("unmarshal transfer in claim error", "err", sdkErr.Error(), "claim", claim)
		return sdk.ClaimResult{}, sdkErr
	}

	transferInSeq := hooks.bridgeKeeper.OracleKeeper.GetCurrentSequence(ctx, types.ClaimTypeTransferIn)

	tokenInfo, err := hooks.bridgeKeeper.TokenMapper.GetToken(ctx, transferInClaim.Symbol)
	if err != nil {
		log.With("module", "bridge").Error("transfer in get token error", "err", err.Error(), "symbol", transferInClaim.Symbol)
		return sdk.ClaimResult{}, sdk.ErrInternal(err.Error())
	}

	if tokenInfo.ContractAddress != transferInClaim.ContractAddress.String() {
		// check decimals of contract
		contractDecimals := hooks.bridgeKeeper.GetContractDecimals(ctx, transferInClaim.ContractAddress)
		if contractDecimals < 0 {
			log.With("module", "bridge").Error("decimals of contract does not exist", "contract_addr", transferInClaim.ContractAddress.String())
			return sdk.ClaimResult{}, types.ErrContractDecimalsNotExists(
				fmt.Sprintf("decimals of contract does not exist, contract_addr=%s",
					transferInClaim.ContractAddress.String()),
			)
		}

		tags, sdkErr := hooks.bridgeKeeper.RefundTransferIn(ctx, contractDecimals, transferInClaim, transferInSeq, types.UnboundToken)
		if sdkErr != nil {
			log.With("module", "bridge").Error("refund transfer in error", "err", sdkErr.Error())
			return sdk.ClaimResult{}, sdkErr
		}
		return sdk.ClaimResult{
			Code: 0,
			Msg:  "",
			Tags: tags,
		}, nil
	}

	if transferInClaim.ExpireTime < ctx.BlockHeader().Time.Unix() {
		tags, sdkErr := hooks.bridgeKeeper.RefundTransferIn(ctx, tokenInfo.ContractDecimals, transferInClaim, transferInSeq, types.Timeout)
		if sdkErr != nil {
			log.With("module", "bridge").Error("refund transfer in error", "err", sdkErr.Error())
			return sdk.ClaimResult{}, sdkErr
		}
		return sdk.ClaimResult{
			Code: 0,
			Msg:  "",
			Tags: tags,
		}, nil
	}

	balance := hooks.bridgeKeeper.BankKeeper.GetCoins(ctx, types.PegAccount)
	var totalTransferInAmount sdk.Coins
	for idx, _ := range transferInClaim.ReceiverAddresses {
		amount := sdk.NewCoin(transferInClaim.Symbol, transferInClaim.Amounts[idx])
		totalTransferInAmount = sdk.Coins{amount}.Plus(totalTransferInAmount)
	}
	if !balance.IsGTE(totalTransferInAmount) {
		tags, sdkErr := hooks.bridgeKeeper.RefundTransferIn(ctx, tokenInfo.ContractDecimals, transferInClaim, transferInSeq, types.InsufficientBalance)
		if sdkErr != nil {
			log.With("module", "bridge").Error("refund transfer in error", "err", sdkErr.Error())
			return sdk.ClaimResult{}, sdkErr
		}
		return sdk.ClaimResult{
			Code: 0,
			Msg:  "",
			Tags: tags,
		}, nil
	}

	for idx, receiverAddr := range transferInClaim.ReceiverAddresses {
		amount := sdk.NewCoin(transferInClaim.Symbol, transferInClaim.Amounts[idx])
		_, sdkErr = hooks.bridgeKeeper.BankKeeper.SendCoins(ctx, types.PegAccount, receiverAddr, sdk.Coins{amount})
		if sdkErr != nil {
			log.With("module", "bridge").Error("send coins error", "err", sdkErr.Error())
			return sdk.ClaimResult{}, sdkErr
		}
	}

	// distribute fee
	relayFee := sdk.Coins{transferInClaim.RelayFee}
	_, _, sdkErr = hooks.bridgeKeeper.BankKeeper.SubtractCoins(ctx, types.PegAccount, relayFee)
	if sdkErr != nil {
		log.With("module", "bridge").Error("subtract coins error", "err", sdkErr.Error())
		return sdk.ClaimResult{}, sdkErr
	}

	if ctx.IsDeliverTx() {
		// add changed accounts
		addressesChanged := append(transferInClaim.ReceiverAddresses, types.PegAccount)
		hooks.bridgeKeeper.Pool.AddAddrs(addressesChanged)

		nextSeq := hooks.bridgeKeeper.OracleKeeper.GetCurrentSequence(ctx, types.ClaimTypeTransferIn)

		// add fee
		fees.Pool.AddAndCommitFee(
			fmt.Sprintf("cross_transfer_in:%d", nextSeq-1),
			sdk.Fee{
				Tokens: relayFee,
				Type:   sdk.FeeForProposer,
			},
		)
	}

	return sdk.ClaimResult{
		Code: 0,
		Msg:  "",
	}, nil
}
