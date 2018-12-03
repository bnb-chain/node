package fees

import (
	"github.com/BiJie/BinanceChain/common/types"
)

// block level pool
var Pool pool

type pool struct {
	fees types.Fee
}

func (p *pool) AddFee(fee types.Fee) {
	p.fees.AddFee(fee)
}

func (p pool) BlockFees() types.Fee {
	return p.fees
}

func (p *pool) Clear() {
	p.fees = types.Fee{}
}
