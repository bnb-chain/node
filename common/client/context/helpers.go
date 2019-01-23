package context

import (
	"github.com/pkg/errors"

	"github.com/tendermint/tendermint/libs/common"
	rpcclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/cosmos/cosmos-sdk/client/context"
)

// QueryWithData queries information about the connected node
func QueryWithData(ctx context.CLIContext, path string, data common.HexBytes) (res []byte, err error) {
	return query(ctx, path, data)
}

// Query from Tendermint with the provided storename and path
func query(ctx context.CLIContext, path string, key common.HexBytes) (res []byte, err error) {
	node, err := ctx.GetNode()
	if err != nil {
		return res, err
	}
	opts := rpcclient.ABCIQueryOptions{
		Height: ctx.Height,
		Prove:  !ctx.TrustNode,
	}
	result, err := node.ABCIQueryWithOptions(path, key, opts)
	if err != nil {
		return res, err
	}
	resp := result.Response
	if resp.Code != uint32(0) {
		return res, errors.Errorf("query failed: (%d) %s", resp.Code, resp.Log)
	}
	return resp.Value, nil
}
