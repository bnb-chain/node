package types

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/binance-chain/node/common/types"
)

const (
	OperateFeeType  = "operate"
	TransferFeeType = "transfer"
	DexFeeType      = "dex"
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
	return nil
}

var _ MsgFeeParams = (*TransferFeeParam)(nil)
type TransferFeeParam struct {
	FixedFeeParams
	MultiTransferFee  int64 `json:"multi_transfer_fee"`
	LowerLimitAsMulti int64 `json:"lower_limit_as_multi"`
}

func (p *TransferFeeParam) GetParamType() string {
	return TransferFeeType
}

func (p *TransferFeeParam) Check() error {
	err := p.FixedFeeParams.Check()
	if err != nil {
		return err
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
	for _, c := range f.FeeParams {
		err := c.Check()
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *FeeChangeParams) Type() string {
	return "fee change param"
}

func (f *FeeChangeParams) String() string {
	bz, err := json.Marshal(f)
	if err != nil {
		return ""
	}
	return string(bz)
}

func NewFeeParam(paramType string, params string) (FeeParam, error) {
	if paramType == OperateFeeType {
		var fixedFeeParam FixedFeeParams
		err := json.Unmarshal([]byte(params), &fixedFeeParam)
		if err != nil {
			return nil, err
		}
		return &fixedFeeParam, nil
	} else if paramType == DexFeeType {
		var dexFeeParam DexFeeParam
		err := json.Unmarshal([]byte(params), &dexFeeParam)
		if err != nil {
			return nil, err
		}
		return &dexFeeParam, nil
	}
	// extend other param type here
	return nil, fmt.Errorf("operate fee type is not found")
}

func (f *FeeChangeParams) Set(value string) error {
	values := strings.Split(value, "/")
	if len(values) != 2 {
		return fmt.Errorf("The fee({param type}/{param map}) is invalid. Length operate-fee is not equal to 2. ")
	}
	paramType := values[0]
	params := values[1]
	feeParam, err := NewFeeParam(paramType, params)
	if err != nil {
		return err
	}
	err = feeParam.Check()
	if err != nil {
		return err
	}
	f.FeeParams = append(f.FeeParams, feeParam)
	return nil
}

// Other params
