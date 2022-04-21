package types

import (
	"sync"
)

// block level pool to avoid frequently call ctx.With harms performance according to our test
//
// NOTE: states keep in this pool should be cleared per-block,
// an appropriate place should be in the end of Commit() with
// deliver state
type Pool struct {
	accounts sync.Map // save tx/gov related addresses (string wrapped bytes) to be published
	txs      sync.Map
}

func (p *Pool) AddTx(tx Tx, txHash string) {
	p.txs.Store(txHash, tx)
}

func (p Pool) GetTxs() sync.Map {
	return p.txs
}

func (p *Pool) AddAddrs(addrs []AccAddress) {
	for _, addr := range addrs {
		p.accounts.Store(string(addr.Bytes()), struct{}{})
	}
}

func (p Pool) TxRelatedAddrs() []string {
	addrs := make([]string, 0, 0)
	p.accounts.Range(func(key, value interface{}) bool {
		addrs = append(addrs, key.(string))
		return true
	})
	return addrs
}

func (p *Pool) Clear() {
	p.accounts = sync.Map{}
	p.txs = sync.Map{}
}
