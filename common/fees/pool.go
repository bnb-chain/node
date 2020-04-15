package fees

import (
	"fmt"
	"sync"

	"github.com/binance-chain/node/common/types"
)

// block level pool
var Pool pool = newPool()

type pool struct {
	fees          map[string]types.Fee // TxHash -> fee
	committedFees types.Fee
	sync.Mutex
}

func newPool() pool {
	return pool{
		fees:          map[string]types.Fee{},
		committedFees: types.Fee{},
	}
}

func (p *pool) AddFee(txHash string, fee types.Fee) {
	p.fees[txHash] = fee
}

func (p *pool) AddAndCommitFee(txHash string, fee types.Fee) {
	p.Lock()
	defer p.Unlock()
	p.fees[txHash] = fee
	p.committedFees.AddFee(fee)
}

func (p *pool) CommitFee(txHash string) {
	if fee, ok := p.fees[txHash]; ok {
		p.committedFees.AddFee(fee)
	} else {
		panic(fmt.Errorf("commit fee for an invalid TxHash(%s)", txHash))
	}
}

func (p pool) BlockFees() types.Fee {
	return p.committedFees
}

func (p *pool) Clear() {
	p.fees = map[string]types.Fee{}
	p.committedFees = types.Fee{}
}

func (p *pool) GetFee(txHash string) *types.Fee {
	if fee, ok := p.fees[txHash]; ok {
		return &fee
	} else {
		return nil
	}
}
