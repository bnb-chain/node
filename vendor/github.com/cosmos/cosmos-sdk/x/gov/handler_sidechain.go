package gov

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/events"
)

func handleMsgSideChainSubmitProposal(ctx sdk.Context, keeper Keeper, msg MsgSideChainSubmitProposal) sdk.Result {
	ctx, err := keeper.ScKeeper.PrepareCtxForSideChain(ctx, msg.SideChainId)
	if err != nil {
		return ErrInvalidSideChainId(keeper.codespace, msg.SideChainId).Result()
	}

	result := handleMsgSubmitProposal(ctx, keeper,
		NewMsgSubmitProposal(msg.Title, msg.Description, msg.ProposalType, msg.Proposer, msg.InitialDeposit,
			msg.VotingPeriod))
	if result.IsOK() {
		result.Tags = result.Tags.AppendTag(events.SideChainID, []byte(msg.SideChainId))
	}
	return result
}

func handleMsgSideChainDeposit(ctx sdk.Context, keeper Keeper, msg MsgSideChainDeposit) sdk.Result {
	ctx, err := keeper.ScKeeper.PrepareCtxForSideChain(ctx, msg.SideChainId)
	if err != nil {
		return ErrInvalidSideChainId(keeper.codespace, msg.SideChainId).Result()
	}

	result := handleMsgDeposit(ctx, keeper, NewMsgDeposit(msg.Depositer, msg.ProposalID, msg.Amount))
	if result.IsOK() {
		result.Tags = result.Tags.AppendTag(events.SideChainID, []byte(msg.SideChainId))
	}
	return result
}

func handleMsgSideChainVote(ctx sdk.Context, keeper Keeper, msg MsgSideChainVote) sdk.Result {
	ctx, err := keeper.ScKeeper.PrepareCtxForSideChain(ctx, msg.SideChainId)
	if err != nil {
		return ErrInvalidSideChainId(keeper.codespace, msg.SideChainId).Result()
	}
	result := handleMsgVote(ctx, keeper, NewMsgVote(msg.Voter, msg.ProposalID, msg.Option))
	if result.IsOK() {
		result.Tags = result.Tags.AppendTag(events.SideChainID, []byte(msg.SideChainId))
	}
	return result
}
