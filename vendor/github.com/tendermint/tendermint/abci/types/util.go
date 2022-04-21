package types

import (
	"bytes"
	"sort"
)

//------------------------------------------------------------------------------

// ValidatorUpdates is a list of validators that implements the Sort interface
type ValidatorUpdates []ValidatorUpdate

var _ sort.Interface = (ValidatorUpdates)(nil)

// All these methods for ValidatorUpdates:
//    Len, Less and Swap
// are for ValidatorUpdates to implement sort.Interface
// which will be used by the sort package.
// See Issue https://github.com/tendermint/abci/issues/212

func (v ValidatorUpdates) Len() int {
	return len(v)
}

// XXX: doesn't distinguish same validator with different power
func (v ValidatorUpdates) Less(i, j int) bool {
	return bytes.Compare(v[i].PubKey.Data, v[j].PubKey.Data) <= 0
}

func (v ValidatorUpdates) Swap(i, j int) {
	v1 := v[i]
	v[i] = v[j]
	v[j] = v1
}

//------------------------------------------------------------------------------

func ConvertDeprecatedDeliverTxResponse(deprecated *ResponseDeliverTxDeprecated) *ResponseDeliverTx {
	if deprecated == nil {
		return nil
	}
	return &ResponseDeliverTx{
		Code:                 deprecated.Code,
		Data:                 deprecated.Data,
		Log:                  deprecated.Log,
		Info:                 deprecated.Info,
		GasWanted:            deprecated.GasWanted,
		GasUsed:              deprecated.GasUsed,
		Events:               []Event{{Attributes: deprecated.Tags}},
		Codespace:            deprecated.Codespace,
		XXX_NoUnkeyedLiteral: deprecated.XXX_NoUnkeyedLiteral,
		XXX_unrecognized:     deprecated.XXX_unrecognized,
		XXX_sizecache:        deprecated.XXX_sizecache,
	}
}

func ConvertDeprecatedBeginBlockResponse(deprecated *ResponseBeginBlockDeprecated) *ResponseBeginBlock {
	if deprecated == nil {
		return nil
	}
	return &ResponseBeginBlock{
		Events:               []Event{{Attributes: deprecated.Tags}},
		XXX_NoUnkeyedLiteral: deprecated.XXX_NoUnkeyedLiteral,
		XXX_unrecognized:     deprecated.XXX_unrecognized,
		XXX_sizecache:        deprecated.XXX_sizecache,
	}
}

func ConvertDeprecatedEndBlockResponse(deprecated *ResponseEndBlockDeprecated) *ResponseEndBlock {
	if deprecated == nil {
		return nil
	}
	return &ResponseEndBlock{
		ValidatorUpdates:      deprecated.ValidatorUpdates,
		ConsensusParamUpdates: deprecated.ConsensusParamUpdates,
		Events:                []Event{{Attributes: deprecated.Tags}},
		XXX_NoUnkeyedLiteral:  deprecated.XXX_NoUnkeyedLiteral,
		XXX_unrecognized:      deprecated.XXX_unrecognized,
		XXX_sizecache:         deprecated.XXX_sizecache,
	}
}
