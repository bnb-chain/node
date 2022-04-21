package slashing

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/bsc"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/sidechain/types"
)

var cdc = codec.New()

// name to identify transaction types
const (
	MsgRoute                 = "slashing"
	TypeMsgUnjail            = "unjail"
	TypeMsgSideChainUnjail   = "side_chain_unjail"
	TypeMsgBscSubmitEvidence = "bsc_submit_evidence"
)

// verify interface at compile time
var _ sdk.Msg = &MsgUnjail{}

// MsgUnjail - struct for unjailing jailed validator
type MsgUnjail struct {
	ValidatorAddr sdk.ValAddress `json:"address"` // address of the validator operator
}

func NewMsgUnjail(validatorAddr sdk.ValAddress) MsgUnjail {
	return MsgUnjail{
		ValidatorAddr: validatorAddr,
	}
}

//nolint
func (msg MsgUnjail) Route() string { return MsgRoute }
func (msg MsgUnjail) Type() string  { return TypeMsgUnjail }
func (msg MsgUnjail) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.AccAddress(msg.ValidatorAddr)}
}

// get the bytes for the message signer to sign on
func (msg MsgUnjail) GetSignBytes() []byte {
	b, err := cdc.MarshalJSON(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

// quick validity check
func (msg MsgUnjail) ValidateBasic() sdk.Error {
	if msg.ValidatorAddr == nil {
		return ErrBadValidatorAddr(DefaultCodespace)
	}
	return nil
}

func (msg MsgUnjail) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

//__________________________________________________________________

// verify interface at compile time
var _ sdk.Msg = &MsgSideChainUnjail{}

// MsgSideChainUnjail - struct for unjailing jailed side chain validator
type MsgSideChainUnjail struct {
	ValidatorAddr sdk.ValAddress `json:"address"` // address of the validator operator
	SideChainId   string         `json:"side_chain_id"`
}

func NewMsgSideChainUnjail(validatorAddr sdk.ValAddress, sideChainId string) MsgSideChainUnjail {
	return MsgSideChainUnjail{
		ValidatorAddr: validatorAddr,
		SideChainId:   sideChainId,
	}
}

//nolint
func (msg MsgSideChainUnjail) Route() string { return MsgRoute }
func (msg MsgSideChainUnjail) Type() string  { return TypeMsgSideChainUnjail }
func (msg MsgSideChainUnjail) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.AccAddress(msg.ValidatorAddr)}
}

// get the bytes for the message signer to sign on
func (msg MsgSideChainUnjail) GetSignBytes() []byte {
	b := MsgCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(b)
}

// quick validity check
func (msg MsgSideChainUnjail) ValidateBasic() sdk.Error {
	if msg.ValidatorAddr == nil {
		return ErrBadValidatorAddr(DefaultCodespace)
	}
	if len(msg.SideChainId) == 0 || len(msg.SideChainId) > types.MaxSideChainIdLength {
		return ErrInvalidInput(DefaultCodespace, fmt.Sprintf("side chain id must be included and max length is %d bytes", types.MaxSideChainIdLength))
	}
	return nil
}

func (msg MsgSideChainUnjail) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

//__________________________________________________________________

// MsgBscSubmitEvidence - struct for submitting evidence for bsc
var _ sdk.Msg = &MsgBscSubmitEvidence{}

type MsgBscSubmitEvidence struct {
	Submitter sdk.AccAddress `json:"submitter"`
	Headers   []bsc.Header   `json:"headers"`
}

func NewMsgBscSubmitEvidence(submitter sdk.AccAddress, headers []bsc.Header) MsgBscSubmitEvidence {
	return MsgBscSubmitEvidence{
		Submitter: submitter,
		Headers:   headers,
	}
}

func (MsgBscSubmitEvidence) Route() string {
	return MsgRoute
}

func (MsgBscSubmitEvidence) Type() string {
	return TypeMsgBscSubmitEvidence
}

func (msg MsgBscSubmitEvidence) ValidateBasic() sdk.Error {
	if len(msg.Submitter) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Expected delegator address length is %d, actual length is %d", sdk.AddrLen, len(msg.Submitter)))
	}
	if len(msg.Headers) != 2 {
		return ErrInvalidEvidence(DefaultCodespace, "Must have 2 headers exactly")
	}
	if err := headerEmptyCheck(msg.Headers[0]); err != nil {
		return err
	}
	if err := headerEmptyCheck(msg.Headers[1]); err != nil {
		return err
	}
	if msg.Headers[0].Number != msg.Headers[1].Number {
		return ErrInvalidEvidence(DefaultCodespace, "The numbers of two block headers are not the same")
	}
	if msg.Headers[0].ParentHash.Cmp(msg.Headers[1].ParentHash) != 0 {
		return ErrInvalidEvidence(DefaultCodespace, "The parent hash of two block headers are not the same")
	}
	signature1, err := msg.Headers[0].GetSignature()
	if err != nil {
		return ErrInvalidEvidence(DefaultCodespace, fmt.Sprintf("Failed to get signature from block header, %s", err.Error()))
	}
	signature2, err := msg.Headers[1].GetSignature()
	if err != nil {
		return ErrInvalidEvidence(DefaultCodespace, fmt.Sprintf("Failed to get signature from block header, %s", err.Error()))
	}
	if bytes.Compare(signature1, signature2) == 0 {
		return ErrInvalidEvidence(DefaultCodespace, "The two blocks are the same")
	}
	return nil
}

func headerEmptyCheck(header bsc.Header) sdk.Error {

	if header.Number == 0 {
		return ErrInvalidEvidence(DefaultCodespace, "header number can not be zero ")
	}
	if header.Difficulty == 0 {
		return ErrInvalidEvidence(DefaultCodespace, "header difficulty can not be zero")
	}
	if header.Extra == nil {
		return ErrInvalidEvidence(DefaultCodespace, "header extra can not be empty")
	}

	return nil
}

func (msg MsgBscSubmitEvidence) GetSignBytes() []byte {
	bz := MsgCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg MsgBscSubmitEvidence) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Submitter}
}

func (msg MsgBscSubmitEvidence) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
