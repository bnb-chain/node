package gov

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// Register concrete types on codec codec
func RegisterCodec(cdc *codec.Codec) {

	cdc.RegisterConcrete(MsgSubmitProposal{}, "cosmos-sdk/MsgSubmitProposal", nil)
	cdc.RegisterConcrete(MsgDeposit{}, "cosmos-sdk/MsgDeposit", nil)
	cdc.RegisterConcrete(MsgVote{}, "cosmos-sdk/MsgVote", nil)

	cdc.RegisterConcrete(MsgSideChainSubmitProposal{}, "cosmos-sdk/MsgSideChainSubmitProposal", nil)
	cdc.RegisterConcrete(MsgSideChainDeposit{}, "cosmos-sdk/MsgSideChainDeposit", nil)
	cdc.RegisterConcrete(MsgSideChainVote{}, "cosmos-sdk/MsgSideChainVote", nil)

	cdc.RegisterInterface((*Proposal)(nil), nil)
	cdc.RegisterConcrete(&TextProposal{}, "gov/TextProposal", nil)
}

var msgCdc = codec.New()
