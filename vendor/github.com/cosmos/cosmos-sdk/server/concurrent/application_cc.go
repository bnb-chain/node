package concurrent

import "github.com/tendermint/tendermint/abci/types"

type ApplicationCC interface {
	types.Application
	PreCheckTx(req types.RequestCheckTx) types.ResponseCheckTx
	PreDeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx
}
