package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	oTypes "github.com/cosmos/cosmos-sdk/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	sType "github.com/cosmos/cosmos-sdk/x/stake/types"

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
		"submit_proposal":          {},
		"deposit":                  {},
		"vote":                     {},
		"dexList":                  {},
		"orderNew":                 {},
		"orderCancel":              {},
		"issueMsg":                 {},
		"mintMsg":                  {},
		"tokensBurn":               {},
		"setAccountFlags":          {},
		"tokensFreeze":             {},
		"create_validator":         {},
		"remove_validator":         {},
		"timeLock":                 {},
		"timeUnlock":               {},
		"timeRelock":               {},
		"crossBind":                {},
		"crossUnbind":              {},
		"crossTransferOut":         {},
		"crossBindRelayFee":        {},
		"crossUnbindRelayFee":      {},
		"crossTransferOutRelayFee": {},
		"oracleClaim":              {},

		"HTLT":        {},
		"depositHTLT": {},
		"claimHTLT":   {},
		"refundHTLT":  {},

		"side_create_validator": {},
		"side_edit_validator":   {},
		"side_delegate":         {},
		"side_redelegate":       {},
		"side_undelegate":       {},

		"bsc_submit_evidence": {},
		"side_chain_unjail":   {},

		"side_submit_proposal": {},
		"side_deposit":         {},
		"side_vote":            {},
		"tinyIssueMsg":        {},
		"miniIssueMsg":        {},
		"miniTokensSetURI":    {},
		"dexListMini":         {},
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
	MsgType string                `json:"msg_type"`
	Fee     int64                 `json:"fee"`
	FeeFor  sdk.FeeDistributeType `json:"fee_for"`
}

func (p *FixedFeeParams) GetParamType() string {
	return OperateFeeType
}

func (p *FixedFeeParams) GetMsgType() string {
	return p.MsgType
}

func (p *FixedFeeParams) Check() error {
	if p.FeeFor != sdk.FeeForProposer && p.FeeFor != sdk.FeeForAll && p.FeeFor != sdk.FeeFree {
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
	if p.FeeFor != sdk.FeeForProposer && p.FeeFor != sdk.FeeForAll && p.FeeFor != sdk.FeeFree {
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

// ---------   Definition cross side chain prams change ------------------- //
type CSCParamChanges struct {
	Changes []CSCParamChange
	ChainID string
}

type CSCParamChange struct {
	Key    string `json:"key"` // the name of the parameter
	Value  string `json:"value"`
	Target string `json:"target"`

	// Since byte slice is not friendly to show in proposal description, omit it.
	ValueBytes  []byte `json:"-"` // the value of the parameter
	TargetBytes []byte `json:"-"` // the address of the target contract
}

func (c *CSCParamChange) Check() error {
	targetBytes, err := hex.DecodeString(c.Target)
	if err != nil {
		return fmt.Errorf("target is not hex encoded, err %v", err)
	}
	c.TargetBytes = targetBytes

	valueBytes, err := hex.DecodeString(c.Value)
	if err != nil {
		return fmt.Errorf("value is not hex encoded, err %v", err)
	}
	c.ValueBytes = valueBytes
	keyBytes := []byte(c.Key)
	if len(keyBytes) <= 0 || len(keyBytes) > math.MaxUint8 {
		return fmt.Errorf("the length of key exceed the limitation")
	}
	if len(c.ValueBytes) <= 0 || len(c.ValueBytes) > math.MaxUint8 {
		return fmt.Errorf("the length of value exceed the limitation")
	}
	if len(c.TargetBytes) != sdk.AddrLen {
		return fmt.Errorf("the length of target address is not %d", sdk.AddrLen)
	}
	return nil
}

//| Proposal type | key length | bytes of  key  | value length | value  | target addr |
//|      1 byte   | 1 byte     |     N bytes    |   1 byte     | M bytes|  20 byte    |
func (c *CSCParamChange) Serialize() []byte {
	keyBytes := []byte(c.Key)
	keyLength := uint8(len(keyBytes))
	value := c.ValueBytes
	valueLength := uint8(len(value))
	totalLen := 3 + sdk.AddrLen + int(keyLength) + int(valueLength)

	bz := make([]byte, totalLen)
	// prepare the proposal type
	start := 0
	bz[start] = 0x00
	start += 1
	bz[start] = keyLength
	start += 1
	copy(bz[start:start+int(keyLength)], keyBytes)
	start += int(keyLength)
	bz[start] = valueLength
	start += 1
	copy(bz[start:start+int(valueLength)], value)
	start += int(valueLength)
	copy(bz[start:start+sdk.AddrLen], c.TargetBytes)
	return bz
}

// ---------   Definition side chain prams change ------------------- //
type SCParam interface {
	Check() error
	Value() interface{}
	// native means weather the parameter stored in native store context or side chain store context
	GetParamAttribute() (paramType string, native bool)
}

type OracleParams struct {
	oTypes.Params
}

func (p *OracleParams) IsNative() bool {
	return true
}

func (p *OracleParams) Value() interface{} {
	return p.Params
}

func (p *OracleParams) GetParamAttribute() (string, bool) {
	return "oracle", true
}

func (p *OracleParams) Check() error {
	return p.Params.UpdateCheck()
}

type StakeParams struct {
	sType.Params
}

func (s *StakeParams) Check() error {
	// only native token support so far, may extend in future.
	if s.BondDenom != types.NativeTokenSymbol {
		return fmt.Errorf("only native token is availabe as bond_denom so far")
	}
	return s.Params.UpdateCheck()
}

func (s *StakeParams) Value() interface{} {
	return s.Params
}

func (s *StakeParams) GetParamAttribute() (string, bool) {
	return "stake", false
}

type SlashParams struct {
	slashing.Params
}

func (s *SlashParams) Check() error {
	return s.Params.UpdateCheck()
}

func (s *SlashParams) Value() interface{} {
	return s.Params
}

func (s *SlashParams) GetParamAttribute() (string, bool) {
	return "slash", false
}

func ToSCParam(p interface{}) SCParam {
	switch iParams := p.(type) {
	case *slashing.Params:
		return &SlashParams{Params: *iParams}
	case *sType.Params:
		return &StakeParams{Params: *iParams}
	case *oTypes.Params:
		return &OracleParams{Params: *iParams}
	default:
		return nil
	}
}

type SCChangeParams struct {
	SCParams    []SCParam `json:"sc_params"`
	Description string    `json:"description"`
}

func (s *SCChangeParams) Check() error {
	supportParams := []SCParam{&SlashParams{}, &OracleParams{}, &StakeParams{}}

	if len(s.SCParams) != len(supportParams) {
		return fmt.Errorf("the sc_params length mismatch, suppose %d", len(supportParams))
	}

	paramSet := make(map[string]bool)
	for _, s := range supportParams {
		stype, _ := s.GetParamAttribute()
		paramSet[stype] = true
	}

	for _, sc := range s.SCParams {
		if sc == nil {
			return fmt.Errorf("sc_params contains empty element")
		}
		err := sc.Check()
		if err != nil {
			return err
		}
		paramType, _ := sc.GetParamAttribute()
		if exist := paramSet[paramType]; exist {
			delete(paramSet, paramType)
		} else {
			return fmt.Errorf("unsupported param type %s", paramType)
		}
	}
	return nil
}
