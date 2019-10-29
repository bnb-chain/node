package pub

import (
	abci "github.com/tendermint/tendermint/abci/types"
)

// block level pool
var Pool = newPool()

type pool struct {
	txResults map[string]abci.ResponseDeliverTx // TxHash -> fee
}

func newPool() pool {
	return pool{
		txResults: make(map[string]abci.ResponseDeliverTx, 0),
	}
}

// Not thread safe.
func (p *pool) AddTxRes(txHash string, txRes abci.ResponseDeliverTx) {
	p.txResults[txHash] = txRes
}

// Not thread safe.
func (p *pool) GetTxRes(txHash string) *abci.ResponseDeliverTx {
	if r, ok := p.txResults[txHash]; ok {
		return &r
	} else {
		return nil
	}
}

// Not thread safe.
func (p *pool) Clean() {
	p.txResults = make(map[string]abci.ResponseDeliverTx, 0)
}
