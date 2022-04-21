package oracle

import (
	"encoding/hex"
	"fmt"
	"runtime/debug"
	"strconv"

	"github.com/cosmos/cosmos-sdk/bsc/rlp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/oracle/types"
	sTypes "github.com/cosmos/cosmos-sdk/x/sidechain/types"
)

func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case types.ClaimMsg:
			return handleClaimMsg(ctx, keeper, msg)
		default:
			errMsg := "Unrecognized oracle msg type"
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleClaimMsg(ctx sdk.Context, oracleKeeper Keeper, msg ClaimMsg) sdk.Result {
	claim := NewClaim(types.GetClaimId(msg.ChainId, types.RelayPackagesChannelId, msg.Sequence),
		sdk.ValAddress(msg.ValidatorAddress), hex.EncodeToString(msg.Payload))

	sequence := oracleKeeper.ScKeeper.GetReceiveSequence(ctx, msg.ChainId, types.RelayPackagesChannelId)
	if sequence != msg.Sequence {
		return types.ErrInvalidSequence(fmt.Sprintf("current sequence of channel %d is %d", types.RelayPackagesChannelId, sequence)).Result()
	}

	prophecy, sdkErr := oracleKeeper.ProcessClaim(ctx, claim)
	if sdkErr != nil {
		return sdkErr.Result()
	}

	if prophecy.Status.Text == types.FailedStatusText {
		oracleKeeper.DeleteProphecy(ctx, prophecy.ID)
		return sdk.Result{}
	}

	if prophecy.Status.Text != types.SuccessStatusText {
		return sdk.Result{}
	}

	packages := types.Packages{}
	err := rlp.DecodeBytes(msg.Payload, &packages)
	if err != nil {
		return types.ErrInvalidPayload("decode packages error").Result()
	}

	events := make([]sdk.Event, 0, len(packages))
	for _, pack := range packages {
		event, sdkErr := handlePackage(ctx, oracleKeeper, msg.ChainId, &pack)
		if sdkErr != nil {
			// only do log, but let reset package get chance to execute.
			ctx.Logger().With("module", "oracle").Error(fmt.Sprintf("process package failed, channel=%d, sequence=%d, error=%v", pack.ChannelId, pack.Sequence, sdkErr))
			return sdkErr.Result()
		} else {
			ctx.Logger().With("module", "oracle").Info(fmt.Sprintf("process package success, channel=%d, sequence=%d", pack.ChannelId, pack.Sequence))
		}
		events = append(events, event)

		// increase channel sequence
		oracleKeeper.ScKeeper.IncrReceiveSequence(ctx, msg.ChainId, pack.ChannelId)
	}

	// delete prophecy when execute claim success
	oracleKeeper.DeleteProphecy(ctx, prophecy.ID)
	oracleKeeper.ScKeeper.IncrReceiveSequence(ctx, msg.ChainId, types.RelayPackagesChannelId)

	return sdk.Result{
		Events: events,
	}
}

func handlePackage(ctx sdk.Context, oracleKeeper Keeper, chainId sdk.ChainID, pack *types.Package) (sdk.Event, sdk.Error) {
	logger := ctx.Logger().With("module", "x/oracle")

	crossChainApp := oracleKeeper.ScKeeper.GetCrossChainApp(ctx, pack.ChannelId)
	if crossChainApp == nil {
		return sdk.Event{}, types.ErrChannelNotRegistered(fmt.Sprintf("channel %d not registered", pack.ChannelId))
	}

	sequence := oracleKeeper.ScKeeper.GetReceiveSequence(ctx, chainId, pack.ChannelId)
	if sequence != pack.Sequence {
		return sdk.Event{}, types.ErrInvalidSequence(fmt.Sprintf("current sequence of channel %d is %d", pack.ChannelId, sequence))
	}

	packageType, relayFee, err := sTypes.DecodePackageHeader(pack.Payload)
	if err != nil {
		return sdk.Event{}, types.ErrInvalidPayloadHeader(err.Error())
	}

	if !sdk.IsValidCrossChainPackageType(packageType) {
		return sdk.Event{}, types.ErrInvalidPackageType()
	}

	feeAmount := relayFee.Int64()
	if feeAmount < 0 {
		return sdk.Event{}, types.ErrFeeOverflow("relayFee overflow")
	}

	fee := sdk.Coins{sdk.Coin{Denom: sdk.NativeTokenSymbol, Amount: feeAmount}}
	_, _, sdkErr := oracleKeeper.BkKeeper.SubtractCoins(ctx, sdk.PegAccount, fee)
	if sdkErr != nil {
		return sdk.Event{}, sdkErr
	}

	if ctx.IsDeliverTx() {
		// add changed accounts
		oracleKeeper.Pool.AddAddrs([]sdk.AccAddress{sdk.PegAccount})

		// add fee
		fees.Pool.AddAndCommitFee(
			fmt.Sprintf("cross_communication:%d:%d:%v", pack.ChannelId, pack.Sequence, packageType),
			sdk.Fee{
				Tokens: fee,
				Type:   sdk.FeeForProposer,
			},
		)
	}

	cacheCtx, write := ctx.CacheContext()
	crash, result := executeClaim(cacheCtx, crossChainApp, pack.Payload, packageType, feeAmount)
	if result.IsOk() {
		write()
	} else if ctx.IsDeliverTx() {
		oracleKeeper.Metrics.ErrNumOfChannels.With("channel_id", fmt.Sprintf("%d", pack.ChannelId)).Add(1)
		destChainName, err := oracleKeeper.ScKeeper.GetDestChainName(chainId)
		if err != nil {
			logger.Error("failed to find name of dest chain", "chainId", chainId)
		} else {
			oracleKeeper.PublishCrossAppFailEvent(ctx, sdk.PegAccount.String(), feeAmount, destChainName)
		}
	}

	// write ack package
	var sendSequence int64 = -1
	if packageType == sdk.SynCrossChainPackageType {
		if crash {
			var ibcErr sdk.Error
			var sendSeq uint64
			if sdk.IsUpgrade(sdk.FixFailAckPackage) && len(pack.Payload) >= sTypes.PackageHeaderLength {
				sendSeq, ibcErr = oracleKeeper.IbcKeeper.CreateRawIBCPackageById(ctx, chainId,
					pack.ChannelId, sdk.FailAckCrossChainPackageType, pack.Payload[sTypes.PackageHeaderLength:])
			} else {
				logger.Error("found payload without header", "channelID", pack.ChannelId, "sequence", pack.Sequence, "payload", hex.EncodeToString(pack.Payload))
				sendSeq, ibcErr = oracleKeeper.IbcKeeper.CreateRawIBCPackageById(ctx, chainId,
					pack.ChannelId, sdk.FailAckCrossChainPackageType, pack.Payload)
			}
			if ibcErr != nil {
				logger.Error("failed to write FailAckCrossChainPackage", "err", err)
				return sdk.Event{}, ibcErr
			}
			sendSequence = int64(sendSeq)
		} else {
			if len(result.Payload) != 0 {
				sendSeq, err := oracleKeeper.IbcKeeper.CreateRawIBCPackageById(ctx, chainId,
					pack.ChannelId, sdk.AckCrossChainPackageType, result.Payload)
				if err != nil {
					logger.Error("failed to write AckCrossChainPackage", "err", err)
					return sdk.Event{}, err
				}
				sendSequence = int64(sendSeq)
			}
		}
	}

	resultTags := sdk.NewTags(
		types.ClaimResultCode, []byte(strconv.FormatInt(int64(result.Code()), 10)),
		types.ClaimResultMsg, []byte(result.Msg()),
		types.ClaimPackageType, []byte(strconv.FormatInt(int64(packageType), 10)),
		// The following tags are for index
		types.ClaimChannel, []byte{uint8(pack.ChannelId)},
		types.ClaimReceiveSequence, []byte(strconv.FormatUint(pack.Sequence, 10)),
	)

	if sendSequence >= 0 {
		resultTags = append(resultTags, sdk.MakeTag(types.ClaimSendSequence, []byte(strconv.FormatInt(sendSequence, 10))))
	}

	if crash {
		resultTags = append(resultTags, sdk.MakeTag(types.ClaimCrash, []byte{1}))
	}

	// emit event if feeAmount is larger than 0
	if feeAmount > 0 {
		resultTags = append(resultTags, sdk.GetPegOutTag(sdk.NativeTokenSymbol, feeAmount))
	}

	if result.Tags != nil {
		resultTags = resultTags.AppendTags(result.Tags)
	}

	event := sdk.Event{
		Type:       types.EventTypeClaim,
		Attributes: resultTags,
	}

	return event, nil
}

func executeClaim(ctx sdk.Context, app sdk.CrossChainApplication, payload []byte, packageType sdk.CrossChainPackageType, relayerFee int64) (crash bool, result sdk.ExecuteResult) {
	defer func() {
		if r := recover(); r != nil {
			log := fmt.Sprintf("recovered: %v\nstack:\n%v", r, string(debug.Stack()))
			logger := ctx.Logger().With("module", "oracle")
			logger.Error("execute claim panic", "err_log", log)
			crash = true
			result = sdk.ExecuteResult{
				Err: sdk.ErrInternal(fmt.Sprintf("execute claim failed: %v", r)),
			}
		}
	}()

	switch packageType {
	case sdk.SynCrossChainPackageType:
		result = app.ExecuteSynPackage(ctx, payload[sTypes.PackageHeaderLength:], relayerFee)
	case sdk.AckCrossChainPackageType:
		result = app.ExecuteAckPackage(ctx, payload[sTypes.PackageHeaderLength:])
	case sdk.FailAckCrossChainPackageType:
		result = app.ExecuteFailAckPackage(ctx, payload[sTypes.PackageHeaderLength:])
	default:
		panic(fmt.Sprintf("receive unexpected package type %d", packageType))
	}
	return
}
