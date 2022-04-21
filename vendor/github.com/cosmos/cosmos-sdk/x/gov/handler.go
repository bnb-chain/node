package gov

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/events"
	"github.com/cosmos/cosmos-sdk/x/gov/tags"
)

// Handle all "gov" type messages.
func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case MsgDeposit:
			return handleMsgDeposit(ctx, keeper, msg)
		case MsgSubmitProposal:
			return handleMsgSubmitProposal(ctx, keeper, msg)
		case MsgVote:
			return handleMsgVote(ctx, keeper, msg)
		case MsgSideChainDeposit:
			return handleMsgSideChainDeposit(ctx, keeper, msg)
		case MsgSideChainSubmitProposal:
			return handleMsgSideChainSubmitProposal(ctx, keeper, msg)
		case MsgSideChainVote:
			return handleMsgSideChainVote(ctx, keeper, msg)
		default:
			errMsg := "Unrecognized gov msg type"
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleMsgSubmitProposal(ctx sdk.Context, keeper Keeper, msg MsgSubmitProposal) sdk.Result {

	proposal := keeper.NewTextProposal(ctx, msg.Title, msg.Description, msg.ProposalType, msg.VotingPeriod)

	hooksErr := keeper.OnProposalSubmitted(ctx, proposal)
	if hooksErr != nil {
		return ErrInvalidProposal(keeper.codespace, hooksErr.Error()).Result()
	}

	proposalID := proposal.GetProposalID()
	proposalIDBytes := []byte(fmt.Sprintf("%d", proposalID))

	err, votingStarted := keeper.AddDeposit(ctx, proposal.GetProposalID(), msg.Proposer, msg.InitialDeposit)
	if err != nil {
		return err.Result()
	}

	resTags := sdk.NewTags(
		tags.Action, tags.ActionSubmitProposal,
		tags.Proposer, []byte(msg.Proposer.String()),
		tags.ProposalID, proposalIDBytes,
	)

	if votingStarted {
		resTags.AppendTag(tags.VotingPeriodStart, proposalIDBytes)
	}

	return sdk.Result{
		Data: proposalIDBytes,
		Tags: resTags,
	}
}

func handleMsgDeposit(ctx sdk.Context, keeper Keeper, msg MsgDeposit) sdk.Result {

	err, votingStarted := keeper.AddDeposit(ctx, msg.ProposalID, msg.Depositer, msg.Amount)
	if err != nil {
		return err.Result()
	}

	proposalIDBytes := keeper.cdc.MustMarshalBinaryBare(msg.ProposalID)

	// TODO: Add tag for if voting period started
	resTags := sdk.NewTags(
		tags.Action, tags.ActionDeposit,
		tags.Depositer, []byte(msg.Depositer.String()),
		tags.ProposalID, proposalIDBytes,
	)

	if votingStarted {
		resTags.AppendTag(tags.VotingPeriodStart, proposalIDBytes)
	}

	return sdk.Result{
		Tags: resTags,
	}
}

func handleMsgVote(ctx sdk.Context, keeper Keeper, msg MsgVote) sdk.Result {
	validator := keeper.vs.Validator(ctx, sdk.ValAddress(msg.Voter))

	if validator == nil {
		return sdk.ErrUnauthorized("Vote is not from a validator operator").Result()
	}

	if validator.GetPower().IsZero() {
		return sdk.ErrUnauthorized("Validator is not bonded").Result()
	}

	err := keeper.AddVote(ctx, msg.ProposalID, msg.Voter, msg.Option)
	if err != nil {
		return err.Result()
	}

	proposalIDBytes := keeper.cdc.MustMarshalBinaryBare(msg.ProposalID)

	resTags := sdk.NewTags(
		tags.Action, tags.ActionVote,
		tags.Voter, []byte(msg.Voter.String()),
		tags.ProposalID, proposalIDBytes,
	)
	return sdk.Result{
		Tags: resTags,
	}
}

type SimpleProposal struct {
	Id      int64
	ChainID string
}

func EndBlocker(baseCtx sdk.Context, keeper Keeper) (refundProposals, notRefundProposals []SimpleProposal) {
	events := sdk.EmptyEvents()
	refundProposals = make([]SimpleProposal, 0)
	notRefundProposals = make([]SimpleProposal, 0)
	chainIDs := []string{NativeChainID}
	contexts := []sdk.Context{baseCtx}
	if sdk.IsUpgrade(sdk.LaunchBscUpgrade) && keeper.ScKeeper != nil {
		tmpSideIDs, storePrefixes := keeper.ScKeeper.GetAllSideChainPrefixes(baseCtx)
		chainIDs = append(chainIDs, tmpSideIDs...)
		for i := range storePrefixes {
			contexts = append(contexts, baseCtx.WithSideChainKeyPrefix(storePrefixes[i]))
		}
	}
	for i := 0; i < len(chainIDs); i++ {
		resEvents, refund, noRefund := settleProposals(contexts[i], keeper, chainIDs[i])
		events = events.AppendEvents(resEvents)
		refundProposals = append(refundProposals, refund...)
		notRefundProposals = append(notRefundProposals, noRefund...)
	}
	baseCtx.EventManager().EmitEvents(events)
	return
}

func settleProposals(ctx sdk.Context, keeper Keeper, chainId string) (resEvents sdk.Events, refundProposals, notRefundProposals []SimpleProposal) {

	logger := ctx.Logger().With("module", "x/gov")

	resEvents = sdk.EmptyEvents()
	refundProposals = make([]SimpleProposal, 0)
	notRefundProposals = make([]SimpleProposal, 0)

	// Delete proposals that haven't met minDeposit
	for ShouldPopInactiveProposalQueue(ctx, keeper) {
		inactiveProposal := keeper.InactiveProposalQueuePop(ctx)
		if inactiveProposal.GetStatus() != StatusDepositPeriod {
			continue
		}
		// distribute deposits to proposer
		keeper.DistributeDeposits(ctx, inactiveProposal.GetProposalID())

		keeper.DeleteProposal(ctx, inactiveProposal)

		notRefundProposals = append(notRefundProposals, SimpleProposal{inactiveProposal.GetProposalID(), chainId})
		event := sdk.NewEvent(events.EventTypeProposalDropped, sdk.NewAttribute(events.ProposalID,
			strconv.FormatInt(inactiveProposal.GetProposalID(), 10)))
		if chainId != NativeChainID {
			event.AppendAttributes(sdk.NewAttribute(events.SideChainID, chainId))
		}
		resEvents = resEvents.AppendEvent(event)

		logger.Info(
			fmt.Sprintf("proposal %d (%s) didn't meet minimum deposit of %v (had only %v); distribute to validator",
				inactiveProposal.GetProposalID(),
				inactiveProposal.GetTitle(),
				keeper.GetDepositParams(ctx).MinDeposit,
				inactiveProposal.GetTotalDeposit(),
			),
		)
	}

	// Check if earliest Active Proposal ended voting period yet
	for ShouldPopActiveProposalQueue(ctx, keeper) {
		activeProposal := keeper.ActiveProposalQueuePop(ctx)

		proposalStartTime := activeProposal.GetVotingStartTime()
		votingPeriod := activeProposal.GetVotingPeriod()
		if ctx.BlockHeader().Time.Before(proposalStartTime.Add(votingPeriod)) {
			continue
		}

		passes, refundDeposits, tallyResults := Tally(ctx, keeper, activeProposal)
		var action string
		if passes {
			activeProposal.SetStatus(StatusPassed)
			action = events.EventTypeProposalPassed

			// refund deposits
			keeper.RefundDeposits(ctx, activeProposal.GetProposalID())
			refundProposals = append(refundProposals, SimpleProposal{activeProposal.GetProposalID(), chainId})
		} else {
			activeProposal.SetStatus(StatusRejected)
			action = events.EventTypeProposalRejected

			// if votes reached quorum and not all votes are abstain, distribute deposits to validator, else refund deposits
			if refundDeposits {
				keeper.RefundDeposits(ctx, activeProposal.GetProposalID())
				refundProposals = append(refundProposals, SimpleProposal{activeProposal.GetProposalID(), chainId})
			} else {
				keeper.DistributeDeposits(ctx, activeProposal.GetProposalID())
				notRefundProposals = append(notRefundProposals, SimpleProposal{activeProposal.GetProposalID(), chainId})
			}
		}

		activeProposal.SetTallyResult(tallyResults)
		keeper.SetProposal(ctx, activeProposal)

		logger.Info(fmt.Sprintf("proposal %d (%s) tallied; passed: %v",
			activeProposal.GetProposalID(), activeProposal.GetTitle(), passes))
		event := sdk.NewEvent(action, sdk.NewAttribute(events.ProposalID,
			strconv.FormatInt(activeProposal.GetProposalID(), 10)))
		if chainId != NativeChainID {
			event.AppendAttributes(sdk.NewAttribute(events.SideChainID, chainId))
		}
		resEvents = resEvents.AppendEvent(event)
	}

	return
}

func ShouldPopInactiveProposalQueue(ctx sdk.Context, keeper Keeper) bool {
	depositParams := keeper.GetDepositParams(ctx)
	peekProposal := keeper.InactiveProposalQueuePeek(ctx)

	if peekProposal == nil {
		return false
	} else if peekProposal.GetStatus() != StatusDepositPeriod {
		return true
	} else if !ctx.BlockHeader().Time.Before(peekProposal.GetSubmitTime().Add(depositParams.MaxDepositPeriod)) {
		return true
	}
	return false
}

func ShouldPopActiveProposalQueue(ctx sdk.Context, keeper Keeper) bool {
	peekProposal := keeper.ActiveProposalQueuePeek(ctx)

	if peekProposal == nil {
		return false
	} else if !ctx.BlockHeader().Time.Before(peekProposal.GetVotingStartTime().Add(peekProposal.GetVotingPeriod())) {
		return true
	}
	return false
}
