package sidechain

import (
	"encoding/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	QuerychannelSettings = "channelSettings"
)

// creates a querier for staking REST endpoints
func NewQuerier(k Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, sdk.Error) {
		switch path[0] {
		case QuerychannelSettings:
			var sideChainId string
			err := k.cdc.UnmarshalJSON(req.Data, &sideChainId)
			if err != nil {
				return nil, ErrInvalidSideChainId(DefaultCodespace, err.Error())
			}
			if len(sideChainId) == 0 {
				return nil, ErrInvalidSideChainId(DefaultCodespace, "SideChainId is missing")
			}
			return queryChannelSettings(ctx, k, sideChainId)
		default:
			return nil, sdk.ErrUnknownRequest("unknown side chain query endpoint")
		}
	}
}

func queryChannelSettings(ctx sdk.Context, k Keeper, sideChainId string) ([]byte, sdk.Error) {
	id, err := k.GetDestChainID(sideChainId)
	if err != nil {
		return nil, ErrInvalidSideChainId(DefaultCodespace, err.Error())
	}
	permissionMap := k.GetChannelSendPermissions(ctx, id)

	res, resErr := json.Marshal(permissionMap)
	if resErr != nil {
		return res, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", resErr.Error()))
	}

	return res, nil
}
