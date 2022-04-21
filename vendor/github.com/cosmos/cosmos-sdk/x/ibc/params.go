package ibc

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/x/params"
)

const (
	DefaultRelayerFeeParam int64 = 1e6 // decimal is 8
	// Default parameter namespace
	DefaultParamspace = "ibc"
)

var (
	ParamRelayerFee = []byte("relayerFee")
)

type Params struct {
	RelayerFee int64 `json:"relayer_fee"`
}

func (p *Params) KeyValuePairs() params.KeyValuePairs {
	return params.KeyValuePairs{
		{ParamRelayerFee, &p.RelayerFee},
	}
}

func (p *Params) UpdateCheck() error {
	if p.RelayerFee <= 0 {
		return fmt.Errorf("the syn_package_fee should be greater than 0")
	}
	return nil
}

func (p *Params) GetParamAttribute() (string, bool) {
	return "ibc", false
}
