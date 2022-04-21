package sidechain

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/sidechain/types"
	"github.com/tendermint/go-amino"
)

//---------------------    ChanPermissionSettingHooks  -----------------
type ChanPermissionSettingHooks struct {
	cdc *amino.Codec
	k   *Keeper
}

func NewChanPermissionSettingHook(cdc *amino.Codec, keeper *Keeper) ChanPermissionSettingHooks {
	return ChanPermissionSettingHooks{cdc, keeper}
}

var _ gov.GovHooks = ChanPermissionSettingHooks{}

func (hooks ChanPermissionSettingHooks) OnProposalSubmitted(ctx sdk.Context, proposal gov.Proposal) error {
	if proposal.GetProposalType() != gov.ProposalTypeManageChanPermission {
		panic(fmt.Sprintf("received wrong type of proposal %x", proposal.GetProposalType()))
	}

	var changeParam types.ChanPermissionSetting
	strProposal := proposal.GetDescription()
	err := hooks.cdc.UnmarshalJSON([]byte(strProposal), &changeParam)
	if err != nil {
		fmt.Errorf("get broken data when unmarshal ChanPermissionSetting msg. proposalId %d, err %v", proposal.GetProposalID(), err)
	}
	if err := changeParam.Check(); err != nil {
		return err
	}
	if _, ok := hooks.k.cfg.destChainNameToID[changeParam.SideChainId]; !ok {
		return fmt.Errorf("the SideChainId do not exist")
	}
	if _, ok := hooks.k.cfg.channelIDToName[changeParam.ChannelId]; !ok {
		return fmt.Errorf("the ChannelId do not exist")
	}
	return nil
}
