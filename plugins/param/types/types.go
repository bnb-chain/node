package types

import (
	"encoding/json"
	"fmt"

	"github.com/binance-chain/node/common/types"
)

const (
	OperateFeeType  = "operate"
	TransferFeeType = "transfer"
	DexFeeType      = "dex"

	JSONFORMAT  = "json"
	AMINOFORMAT = "amino"
)

var (
	// To avoid cycle import , use literal key. Please update here when new type message is introduced.
	ValidFixedFeeMsgTypes = map[string]struct{}{
		"submit_proposal":  {},
		"deposit":          {},
		"vote":             {},
		"dexList":          {},
		"orderNew":         {},
		"orderCancel":      {},
		"issueMsg":         {},
		"mintMsg":          {},
		"tokensBurn":       {},
		"setAccountFlags":  {},
		"tokensFreeze":     {},
		"create_validator": {},
		"remove_validator": {},
		"timeLock":         {},
		"timeUnlock":       {},
		"timeRelock":       {},

		"HTLT":        {},
		"depositHTLT": {},
		"claimHTLT":   {},
		"refundHTLT":  {},

		"side_create_validator": {},
		"side_edit_validator":   {},
		"side_delegate":         {},
		"side_redelegate":       {},
		"side_undelegate":       {},

		"submit_side_chain_evidence": {},
		"side_chain_unjail":          {},
	}

	ValidTransferFeeMsgTypes = map[string]struct{}{
		"send": {},
	}
)

type LastProposalID struct {
	ProposalID int64 `json:"proposal_id"`
}

type GenesisState struct {
	FeeGenesis []FeeParam `json:"fees"`
}

// ---------   Definition about fee prams  ------------------- //

type FeeChangeParams struct {
	FeeParams   []FeeParam `json:"fee_params"`
	Description string     `json:"description"`
}

type FeeParam interface {
	GetParamType() string
	Check() error
}

var _ FeeParam = MsgFeeParams(nil)

type MsgFeeParams interface {
	FeeParam
	GetMsgType() string
}

var _ MsgFeeParams = (*FixedFeeParams)(nil)

type FixedFeeParams struct {
	MsgType string                  `json:"msg_type"`
	Fee     int64                   `json:"fee"`
	FeeFor  types.FeeDistributeType `json:"fee_for"`
}

func (p *FixedFeeParams) GetParamType() string {
	return OperateFeeType
}

func (p *FixedFeeParams) GetMsgType() string {
	return p.MsgType
}

func (p *FixedFeeParams) Check() error {
	if p.FeeFor != types.FeeForProposer && p.FeeFor != types.FeeForAll && p.FeeFor != types.FeeFree {
		return fmt.Errorf("fee_for %d is invalid", p.FeeFor)
	}
	if p.Fee < 0 {
		return fmt.Errorf("fee(%d) should not be negative", p.Fee)
	}
	if _, ok := ValidFixedFeeMsgTypes[p.GetMsgType()]; !ok {
		return fmt.Errorf("msg type %s can't be fixedFeeParams", p.GetMsgType())
	}
	return nil
}

var _ MsgFeeParams = (*TransferFeeParam)(nil)

type TransferFeeParam struct {
	FixedFeeParams    `json:"fixed_fee_params"`
	MultiTransferFee  int64 `json:"multi_transfer_fee"`
	LowerLimitAsMulti int64 `json:"lower_limit_as_multi"`
}

func (p *TransferFeeParam) GetParamType() string {
	return TransferFeeType
}

func (p *TransferFeeParam) Check() error {
	if p.FeeFor != types.FeeForProposer && p.FeeFor != types.FeeForAll && p.FeeFor != types.FeeFree {
		return fmt.Errorf("fee_for %d is invalid", p.FeeFor)
	}
	if p.Fee <= 0 || p.MultiTransferFee <= 0 {
		return fmt.Errorf("both fee(%d) and multi_transfer_fee(%d) should be positive", p.Fee, p.MultiTransferFee)
	}
	if p.MultiTransferFee > p.Fee {
		return fmt.Errorf("multi_transfer_fee(%d) should not be bigger than fee(%d)", p.MultiTransferFee, p.Fee)
	}
	if p.LowerLimitAsMulti <= 1 {
		return fmt.Errorf("lower_limit_as_multi should > 1")
	}
	if _, ok := ValidTransferFeeMsgTypes[p.GetMsgType()]; !ok {
		return fmt.Errorf("msg type %s can't be transferFeeParam", p.GetMsgType())
	}
	return nil
}

type DexFeeField struct {
	FeeName  string `json:"fee_name"`
	FeeValue int64  `json:"fee_value"`
}

type DexFeeParam struct {
	DexFeeFields []DexFeeField `json:"dex_fee_fields"`
}

func (p *DexFeeParam) isNil() bool {
	for _, d := range p.DexFeeFields {
		if d.FeeValue < 0 {
			return true
		}
	}
	return false
}

func (p *DexFeeParam) GetParamType() string {
	return DexFeeType
}

func (p *DexFeeParam) Check() error {
	if p.isNil() {
		return fmt.Errorf("Dex fee param is less than 0 ")
	}
	return nil
}

func (f *FeeChangeParams) Check() error {
	return checkFeeParams(f.FeeParams)
}

func (f *FeeChangeParams) String() string {
	bz, err := json.Marshal(f)
	if err != nil {
		return ""
	}
	return string(bz)
}

func checkFeeParams(fees []FeeParam) error {
	numDexFeeParams := 0
	for _, c := range fees {
		err := c.Check()
		if err != nil {
			return err
		}
		if _, ok := c.(*DexFeeParam); ok {
			numDexFeeParams++
		}
	}
	if numDexFeeParams > 1 {
		return fmt.Errorf("have more than one DexFeeParam, actural %d", numDexFeeParams)
	}
	return nil
}
