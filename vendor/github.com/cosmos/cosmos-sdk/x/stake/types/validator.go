package types

import (
	"bytes"
	"fmt"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Validator defines the total amount of bond shares and their exchange rate to
// coins. Accumulation of interest is modelled as an in increase in the
// exchange rate, and slashing as a decrease.  When coins are delegated to this
// validator, the validator is credited with a Delegation whose number of
// bond shares is based on the amount of coins delegated divided by the current
// exchange rate. Voting power can be calculated as total bonds multiplied by
// exchange rate.
type Validator struct {
	FeeAddr      sdk.AccAddress `json:"fee_addr"`                   // address for fee collection
	OperatorAddr sdk.ValAddress `json:"operator_address"`           // address of the validator's operator; bech encoded in JSON
	ConsPubKey   crypto.PubKey  `json:"consensus_pubkey,omitempty"` // the consensus public key of the validator; bech encoded in JSON
	Jailed       bool           `json:"jailed"`                     // has the validator been jailed from bonded status?

	Status          sdk.BondStatus `json:"status"`           // validator status (bonded/unbonding/unbonded)
	Tokens          sdk.Dec        `json:"tokens"`           // delegated tokens (incl. self-delegation)
	DelegatorShares sdk.Dec        `json:"delegator_shares"` // total shares issued to a validator's delegators

	Description        Description `json:"description"`           // description terms for the validator
	BondHeight         int64       `json:"bond_height"`           // earliest height as a bonded validator
	BondIntraTxCounter int16       `json:"bond_intra_tx_counter"` // block-local tx index of validator change

	UnbondingHeight  int64     `json:"unbonding_height"` // if unbonding, height at which this validator has begun unbonding
	UnbondingMinTime time.Time `json:"unbonding_time"`   // if unbonding, min time for the validator to complete unbonding

	Commission Commission `json:"commission"` // commission parameters

	DistributionAddr sdk.AccAddress `json:"distribution_addr,omitempty"` // the address receives rewards from the side address, and distribute rewards to delegators. It's auto generated
	SideChainId      string         `json:"side_chain_id,omitempty"`     // side chain id to distinguish different side chains
	SideConsAddr     []byte         `json:"side_cons_addr,omitempty"`    // consensus address of the side chain validator, this replaces the `ConsPubKey`
	SideFeeAddr      []byte         `json:"side_fee_addr,omitempty"`     // fee address on the side chain
}

// NewValidator - initialize a new validator
func NewValidator(operator sdk.ValAddress, pubKey crypto.PubKey, description Description) Validator {
	return NewValidatorWithFeeAddr(sdk.AccAddress(operator), operator, pubKey, description)
}

// Note a few fields are initialized with default value. They will be updated later
func NewValidatorWithFeeAddr(feeAddr sdk.AccAddress, operator sdk.ValAddress, pubKey crypto.PubKey, description Description) Validator {
	return Validator{
		FeeAddr:            feeAddr,
		OperatorAddr:       operator,
		ConsPubKey:         pubKey,
		Jailed:             false,
		Status:             sdk.Unbonded,
		Tokens:             sdk.ZeroDec(),
		DelegatorShares:    sdk.ZeroDec(),
		Description:        description,
		BondHeight:         int64(0),
		BondIntraTxCounter: int16(0),
		UnbondingHeight:    int64(0),
		UnbondingMinTime:   time.Unix(0, 0).UTC(),
		Commission:         NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
	}
}

func NewSideChainValidator(feeAddr sdk.AccAddress, operator sdk.ValAddress, description Description, sideChainId string, sideConsAddr, sideFeeAddr []byte) Validator {
	return Validator{
		FeeAddr:            feeAddr,
		OperatorAddr:       operator,
		ConsPubKey:         nil, // side chain validators do not need this
		Jailed:             false,
		Status:             sdk.Unbonded,
		Tokens:             sdk.ZeroDec(),
		DelegatorShares:    sdk.ZeroDec(),
		Description:        description,
		BondHeight:         int64(0),
		BondIntraTxCounter: int16(0),
		UnbondingMinTime:   time.Unix(0, 0).UTC(),
		UnbondingHeight:    int64(0),
		Commission:         NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
		DistributionAddr:   generateDistributionAddr(operator, sideChainId),
		SideChainId:        sideChainId,
		SideConsAddr:       sideConsAddr,
		SideFeeAddr:        sideFeeAddr,
	}
}

func generateDistributionAddr(operator sdk.ValAddress, sideChainId string) sdk.AccAddress {
	// DistributionAddr = hash(sideChainId) ^ operator
	// so we can easily recover operator address from DistributionAddr,
	// operator = DistributionAddr ^ hash(sideChainId)
	return sdk.XOR(tmhash.SumTruncated([]byte(sideChainId)), operator)
}

// return the redelegation without fields contained within the key for the store
func MustMarshalValidator(cdc *codec.Codec, validator Validator) []byte {
	return cdc.MustMarshalBinaryLengthPrefixed(validator)
}

// unmarshal a redelegation from a store key and value
func MustUnmarshalValidator(cdc *codec.Codec, value []byte) Validator {
	validator, err := UnmarshalValidator(cdc, value)
	if err != nil {
		panic(err)
	}
	return validator
}

func UnmarshalValidator(cdc *codec.Codec, value []byte) (validator Validator, err error) {
	err = cdc.UnmarshalBinaryLengthPrefixed(value, &validator)
	return validator, err
}

func MustMarshalValidators(cdc *codec.Codec, validators []Validator) []byte {
	return cdc.MustMarshalBinaryLengthPrefixed(validators)
}

func MustUnmarshalValidators(cdc *codec.Codec, value []byte) []Validator {
	validators, err := UnmarshalValidators(cdc, value)
	if err != nil {
		panic(err)
	}
	return validators
}

func UnmarshalValidators(cdc *codec.Codec, value []byte) (validators []Validator, err error) {
	err = cdc.UnmarshalBinaryLengthPrefixed(value, &validators)
	return validators, err
}

// HumanReadableString returns a human readable string representation of a
// validator. An error is returned if the operator or the operator's public key
// cannot be converted to Bech32 format.
func (v Validator) HumanReadableString() (string, error) {
	var bechConsPubKey string
	var err error
	if v.ConsPubKey != nil {
		bechConsPubKey, err = sdk.Bech32ifyConsPub(v.ConsPubKey)
		if err != nil {
			return "", err
		}
	}
	resp := "Validator \n"
	resp += fmt.Sprintf("Fee Address: %s\n", v.FeeAddr)
	resp += fmt.Sprintf("Operator Address: %s\n", v.OperatorAddr)
	resp += fmt.Sprintf("Validator Consensus Pubkey: %s\n", bechConsPubKey)
	resp += fmt.Sprintf("Jailed: %v\n", v.Jailed)
	resp += fmt.Sprintf("Status: %s\n", sdk.BondStatusToString(v.Status))
	resp += fmt.Sprintf("Tokens: %s\n", v.Tokens)
	resp += fmt.Sprintf("Delegator Shares: %s\n", v.DelegatorShares)
	resp += fmt.Sprintf("Description: %s\n", v.Description)
	resp += fmt.Sprintf("Bond Height: %d\n", v.BondHeight)
	resp += fmt.Sprintf("Unbonding Height: %d\n", v.UnbondingHeight)
	resp += fmt.Sprintf("Minimum Unbonding Time: %v\n", v.UnbondingMinTime)
	resp += fmt.Sprintf("Commission: {%s}\n", v.Commission)
	if len(v.SideChainId) != 0 {
		resp += fmt.Sprintf("Distribution Addr: %s\n", v.DistributionAddr)
		resp += fmt.Sprintf("Side Chain Id: %s\n", v.SideChainId)
		resp += fmt.Sprintf("Consensus Addr on Side Chain: %s\n", sdk.HexAddress(v.SideConsAddr))
		resp += fmt.Sprintf("Fee Addr on Side Chain: %s\n", sdk.HexAddress(v.SideFeeAddr))
	}

	return resp, nil
}

//___________________________________________________________________

// this is a helper struct used for JSON de- and encoding only
type bechValidator struct {
	FeeAddr      sdk.AccAddress `json:"fee_addr"`                   // the bech32 address for fee collection
	OperatorAddr sdk.ValAddress `json:"operator_address"`           // the bech32 address of the validator's operator
	ConsPubKey   string         `json:"consensus_pubkey,omitempty"` // the bech32 consensus public key of the validator
	Jailed       bool           `json:"jailed"`                     // has the validator been jailed from bonded status?

	Status          sdk.BondStatus `json:"status"`           // validator status (bonded/unbonding/unbonded)
	Tokens          sdk.Dec        `json:"tokens"`           // delegated tokens (incl. self-delegation)
	DelegatorShares sdk.Dec        `json:"delegator_shares"` // total shares issued to a validator's delegators

	Description        Description `json:"description"`           // description terms for the validator
	BondHeight         int64       `json:"bond_height"`           // earliest height as a bonded validator
	BondIntraTxCounter int16       `json:"bond_intra_tx_counter"` // block-local tx index of validator change

	UnbondingHeight  int64     `json:"unbonding_height"` // if unbonding, height at which this validator has begun unbonding
	UnbondingMinTime time.Time `json:"unbonding_time"`   // if unbonding, min time for the validator to complete unbonding

	Commission Commission `json:"commission"` // commission parameters

	DistributionAddr sdk.AccAddress `json:"distribution_addr,omitempty"` // the address receives rewards from the side address, and distribute rewards to delegators. It's auto generated
	SideChainId      string         `json:"side_chain_id,omitempty"`     // side chain id to distinguish different side chains
	SideConsAddr     string         `json:"side_cons_addr,omitempty"`    // consensus address of the side chain validator, this replaces the `ConsPubKey`
	SideFeeAddr      string         `json:"side_fee_addr,omitempty"`     // fee address on the side chain
}

// MarshalJSON marshals the validator to JSON using Bech32
func (v Validator) MarshalJSON() ([]byte, error) {
	var bechConsPubKey string
	var err error
	if v.ConsPubKey != nil {
		bechConsPubKey, err = sdk.Bech32ifyConsPub(v.ConsPubKey)
		if err != nil {
			return nil, err
		}
	}

	return codec.Cdc.MarshalJSON(bechValidator{
		FeeAddr:            v.FeeAddr,
		OperatorAddr:       v.OperatorAddr,
		ConsPubKey:         bechConsPubKey,
		Jailed:             v.Jailed,
		Status:             v.Status,
		Tokens:             v.Tokens,
		DelegatorShares:    v.DelegatorShares,
		Description:        v.Description,
		BondHeight:         v.BondHeight,
		BondIntraTxCounter: v.BondIntraTxCounter,
		UnbondingHeight:    v.UnbondingHeight,
		UnbondingMinTime:   v.UnbondingMinTime,
		Commission:         v.Commission,
		DistributionAddr:   v.DistributionAddr,
		SideChainId:        v.SideChainId,
		SideConsAddr:       sdk.HexAddress(v.SideConsAddr),
		SideFeeAddr:        sdk.HexAddress(v.SideFeeAddr),
	})
}

// UnmarshalJSON unmarshals the validator from JSON using Bech32
func (v *Validator) UnmarshalJSON(data []byte) error {
	bv := &bechValidator{}
	if err := codec.Cdc.UnmarshalJSON(data, bv); err != nil {
		return err
	}
	var consPubKey crypto.PubKey
	if len(bv.ConsPubKey) != 0 {
		getConsPubKey, err := sdk.GetConsPubKeyBech32(bv.ConsPubKey)
		if err != nil {
			return err
		}
		consPubKey = getConsPubKey
	}

	*v = Validator{
		FeeAddr:            bv.FeeAddr,
		OperatorAddr:       bv.OperatorAddr,
		ConsPubKey:         consPubKey,
		Jailed:             bv.Jailed,
		Tokens:             bv.Tokens,
		Status:             bv.Status,
		DelegatorShares:    bv.DelegatorShares,
		Description:        bv.Description,
		BondHeight:         bv.BondHeight,
		BondIntraTxCounter: bv.BondIntraTxCounter,
		UnbondingHeight:    bv.UnbondingHeight,
		UnbondingMinTime:   bv.UnbondingMinTime,
		Commission:         bv.Commission,
	}
	if len(bv.SideChainId) != 0 {
		v.DistributionAddr = bv.DistributionAddr
		v.SideChainId = bv.SideChainId
		if sideConsAddr, err := sdk.HexDecode(bv.SideConsAddr); err != nil {
			return err
		} else {
			v.SideConsAddr = sideConsAddr
		}
		if sideFeeAddr, err := sdk.HexDecode(bv.SideFeeAddr); err != nil {
			return err
		} else {
			v.SideFeeAddr = sideFeeAddr
		}
	}

	return nil
}

//___________________________________________________________________

// only the vitals - does not check bond height of IntraTxCounter
func (v Validator) Equal(v2 Validator) bool {
	return v.FeeAddr.Equals(v2.FeeAddr) &&
		v.ConsPubKey.Equals(v2.ConsPubKey) &&
		v.OperatorAddr.Equals(v2.OperatorAddr) &&
		v.Status.Equal(v2.Status) &&
		v.Tokens.Equal(v2.Tokens) &&
		v.DelegatorShares.Equal(v2.DelegatorShares) &&
		v.Description.Equals(v2.Description) &&
		v.Commission.Equal(v2.Commission) &&
		v.SideChainId == v2.SideChainId &&
		v.DistributionAddr.Equals(v2.DistributionAddr) &&
		bytes.Equal(v.SideConsAddr, v2.SideConsAddr) &&
		bytes.Equal(v.SideFeeAddr, v2.SideFeeAddr)
}

// return the TM validator address
func (v Validator) ConsAddress() sdk.ConsAddress {
	return sdk.ConsAddress(v.ConsPubKey.Address())
}

// constant used in flags to indicate that description field should not be updated
const DoNotModifyDesc = "[do-not-modify]"

// Description - description fields for a validator
type Description struct {
	Moniker  string `json:"moniker"`  // name
	Identity string `json:"identity"` // optional identity signature (ex. UPort or Keybase)
	Website  string `json:"website"`  // optional website link
	Details  string `json:"details"`  // optional details
}

// NewDescription returns a new Description with the provided values.
func NewDescription(moniker, identity, website, details string) Description {
	return Description{
		Moniker:  moniker,
		Identity: identity,
		Website:  website,
		Details:  details,
	}
}

// UpdateDescription updates the fields of a given description. An error is
// returned if the resulting description contains an invalid length.
func (d Description) UpdateDescription(d2 Description) (Description, sdk.Error) {
	if d2.Moniker == DoNotModifyDesc {
		d2.Moniker = d.Moniker
	}
	if d2.Identity == DoNotModifyDesc {
		d2.Identity = d.Identity
	}
	if d2.Website == DoNotModifyDesc {
		d2.Website = d.Website
	}
	if d2.Details == DoNotModifyDesc {
		d2.Details = d.Details
	}

	return Description{
		Moniker:  d2.Moniker,
		Identity: d2.Identity,
		Website:  d2.Website,
		Details:  d2.Details,
	}.EnsureLength()
}

func (d Description) Equals(d2 Description) bool {
	return d.Details == d2.Details &&
		d.Identity == d2.Identity &&
		d.Moniker == d2.Moniker &&
		d.Website == d2.Website
}

// EnsureLength ensures the length of a validator's description.
func (d Description) EnsureLength() (Description, sdk.Error) {
	if len(d.Moniker) == 0 {
		return d, ErrEmptyMoniker(DefaultCodespace)
	}
	if len(d.Moniker) > 70 {
		return d, ErrDescriptionLength(DefaultCodespace, "moniker", len(d.Moniker), 70)
	}
	if len(d.Identity) > 3000 {
		return d, ErrDescriptionLength(DefaultCodespace, "identity", len(d.Identity), 3000)
	}
	if len(d.Website) > 140 {
		return d, ErrDescriptionLength(DefaultCodespace, "website", len(d.Website), 140)
	}
	if len(d.Details) > 280 {
		return d, ErrDescriptionLength(DefaultCodespace, "details", len(d.Details), 280)
	}

	return d, nil
}

// ABCIValidatorUpdate returns an abci.ValidatorUpdate from a staked validator type
// with the full validator power
func (v Validator) ABCIValidatorUpdate() abci.ValidatorUpdate {
	return abci.ValidatorUpdate{
		PubKey: tmtypes.TM2PB.PubKey(v.ConsPubKey),
		Power:  v.BondedTokens().RawInt(),
	}
}

// ABCIValidatorUpdateZero returns an abci.ValidatorUpdate from a staked validator type
// with zero power used for validator updates.
func (v Validator) ABCIValidatorUpdateZero() abci.ValidatorUpdate {
	return abci.ValidatorUpdate{
		PubKey: tmtypes.TM2PB.PubKey(v.ConsPubKey),
		Power:  0,
	}
}

// UpdateStatus updates the location of the shares within a validator
// to reflect the new status
func (v Validator) UpdateStatus(pool Pool, NewStatus sdk.BondStatus) (Validator, Pool) {

	switch v.Status {
	case sdk.Unbonded:

		switch NewStatus {
		case sdk.Unbonded:
			return v, pool
		case sdk.Bonded:
			pool = pool.looseTokensToBonded(v.Tokens)
		}
	case sdk.Unbonding:

		switch NewStatus {
		case sdk.Unbonding:
			return v, pool
		case sdk.Bonded:
			pool = pool.looseTokensToBonded(v.Tokens)
		}
	case sdk.Bonded:

		switch NewStatus {
		case sdk.Bonded:
			return v, pool
		default:
			pool = pool.bondedTokensToLoose(v.Tokens)
		}
	}

	v.Status = NewStatus
	return v, pool
}

// calculate the token worth of provided shares
func (v Validator) TokensFromShares(shares sdk.Dec) sdk.Dec {
	if v.DelegatorShares.IsZero() {
		return sdk.ZeroDec()
	}
	result, err := sdk.MulQuoDec(shares, v.Tokens, v.DelegatorShares)
	if err != nil {
		panic(err)
	}
	return result
}

// SharesFromTokens returns the shares of a delegation given a bond amount. It
// returns an error if the validator has no tokens.
func (v Validator) SharesFromTokens(amt sdk.Dec) sdk.Dec {
	if v.Tokens.IsZero() {
		return sdk.ZeroDec()
	}
	result, err := sdk.MulQuoDec(v.DelegatorShares, amt, v.Tokens)
	if err != nil {
		panic(err)
	}
	return result
}

// removes tokens from a validator
func (v Validator) RemoveTokens(pool Pool, tokens sdk.Dec) (Validator, Pool) {
	if v.Status == sdk.Bonded {
		pool = pool.bondedTokensToLoose(tokens)
	}

	v.Tokens = v.Tokens.Sub(tokens)
	return v, pool
}

// SetInitialCommission attempts to set a validator's initial commission. An
// error is returned if the commission is invalid.
func (v Validator) SetInitialCommission(commission Commission) (Validator, sdk.Error) {
	if err := commission.Validate(); err != nil {
		return v, err
	}

	v.Commission = commission
	return v, nil
}

//_________________________________________________________________________________________________________

// AddTokensFromDel adds tokens to a validator
func (v Validator) AddTokensFromDel(pool Pool, amount int64) (Validator, Pool, sdk.Dec) {

	// bondedShare/delegatedShare
	amountDec := sdk.NewDecFromInt(amount)

	if v.Status == sdk.Bonded {
		pool = pool.looseTokensToBonded(amountDec)
	}

	var issuedShares sdk.Dec
	if v.DelegatorShares.IsZero() {
		// the first delegation to a validator sets the exchange rate to one
		issuedShares = amountDec
	} else {
		shares := v.SharesFromTokens(amountDec)
		issuedShares = shares
	}
	v.Tokens = v.Tokens.Add(amountDec)
	v.DelegatorShares = v.DelegatorShares.Add(issuedShares)

	return v, pool, issuedShares
}

// RemoveDelShares removes delegator shares from a validator.
func (v Validator) RemoveDelShares(pool Pool, delShares sdk.Dec) (Validator, Pool, sdk.Dec) {
	remainingShares := v.DelegatorShares.Sub(delShares)

	var issuedTokens sdk.Dec
	if remainingShares.IsZero() {
		// last delegation share gets any trimmings
		issuedTokens = v.Tokens
		v.Tokens = sdk.ZeroDec()
	} else {
		issuedTokens = v.TokensFromShares(delShares)
		v.Tokens = v.Tokens.Sub(issuedTokens)
	}

	v.DelegatorShares = remainingShares

	if v.Status == sdk.Bonded {
		pool = pool.bondedTokensToLoose(issuedTokens)
	}

	return v, pool, issuedTokens
}

// DelegatorShareExRate gets the exchange rate of tokens over delegator shares.
// UNITS: tokens/delegator-shares
func (v Validator) DelegatorShareExRate() sdk.Dec {
	if v.DelegatorShares.IsZero() {
		return sdk.OneDec()
	}
	return v.Tokens.Quo(v.DelegatorShares)
}

// Get the bonded tokens which the validator holds
func (v Validator) BondedTokens() sdk.Dec {
	if v.Status == sdk.Bonded {
		return v.Tokens
	}
	return sdk.ZeroDec()
}

// IsBonded checks if the validator status equals Bonded
func (v Validator) IsBonded() bool {
	return v.GetStatus().Equal(sdk.Bonded)
}

// IsUnbonded checks if the validator status equals Unbonded
func (v Validator) IsUnbonded() bool {
	return v.GetStatus().Equal(sdk.Unbonded)
}

// IsUnbonding checks if the validator status equals Unbonding
func (v Validator) IsUnbonding() bool {
	return v.GetStatus().Equal(sdk.Unbonding)
}

//______________________________________________________________________

// ensure fulfills the sdk validator types
var _ sdk.Validator = Validator{}

// nolint - for sdk.Validator
func (v Validator) GetJailed() bool              { return v.Jailed }
func (v Validator) GetMoniker() string           { return v.Description.Moniker }
func (v Validator) GetStatus() sdk.BondStatus    { return v.Status }
func (v Validator) GetFeeAddr() sdk.AccAddress   { return v.FeeAddr }
func (v Validator) GetOperator() sdk.ValAddress  { return v.OperatorAddr }
func (v Validator) GetConsPubKey() crypto.PubKey { return v.ConsPubKey }
func (v Validator) GetConsAddr() sdk.ConsAddress { return sdk.ConsAddress(v.ConsPubKey.Address()) }
func (v Validator) GetPower() sdk.Dec            { return v.BondedTokens() }
func (v Validator) GetTokens() sdk.Dec           { return v.Tokens }
func (v Validator) GetCommission() sdk.Dec       { return v.Commission.Rate }
func (v Validator) GetDelegatorShares() sdk.Dec  { return v.DelegatorShares }
func (v Validator) GetBondHeight() int64         { return v.BondHeight }
func (v Validator) GetSideChainConsAddr() []byte { return v.SideConsAddr }
func (v Validator) IsSideChainValidator() bool   { return len(v.SideChainId) != 0 }

func (v Validator) IsSelfDelegator(address sdk.AccAddress) bool { return v.FeeAddr.Equals(address) }
