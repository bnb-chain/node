package types

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ClaimTypeSkipSequence      sdk.ClaimType = 0x1
	ClaimTypeUpdateBind        sdk.ClaimType = 0x2
	ClaimTypeTransferOutRefund sdk.ClaimType = 0x3
	ClaimTypeTransferIn        sdk.ClaimType = 0x4

	ClaimTypeSkipSequenceName      = "SkipSequence"
	ClaimTypeUpdateBindName        = "UpdateBind"
	ClaimTypeTransferOutRefundName = "TransferOutRefund"
	ClaimTypeTransferInName        = "TransferIn"
)

type TransferInClaim struct {
	ContractAddress   SmartChainAddress   `json:"contract_address"`
	RefundAddresses   []SmartChainAddress `json:"refund_addresses"`
	ReceiverAddresses []sdk.AccAddress    `json:"receiver_addresses"`
	Amounts           []int64             `json:"amounts"`
	Symbol            string              `json:"symbol"`
	RelayFee          sdk.Coin            `json:"relay_fee"`
	ExpireTime        int64               `json:"expire_time"`
}

func GetTransferInClaimFromOracleClaim(claim string) (TransferInClaim, sdk.Error) {
	transferClaim := TransferInClaim{}
	err := json.Unmarshal([]byte(claim), &transferClaim)
	if err != nil {
		return TransferInClaim{}, ErrInvalidTransferMsg(err.Error())
	}
	return transferClaim, nil
}

type TransferOutRefundClaim struct {
	RefundAddress sdk.AccAddress `json:"refund_address"`
	Amount        sdk.Coin       `json:"amount"`
	RefundReason  RefundReason   `json:"refund_reason"`
}

func GetTransferOutRefundClaimFromOracleClaim(claim string) (TransferOutRefundClaim, sdk.Error) {
	refundClaim := TransferOutRefundClaim{}
	err := json.Unmarshal([]byte(claim), &refundClaim)
	if err != nil {
		return TransferOutRefundClaim{}, ErrInvalidTransferMsg(err.Error())
	}
	return refundClaim, nil
}

type UpdateBindClaim struct {
	Status          BindStatus        `json:"status"`
	Symbol          string            `json:"symbol"`
	ContractAddress SmartChainAddress `json:"contract_address"`
}

func GetUpdateBindClaimFromOracleClaim(claim string) (UpdateBindClaim, sdk.Error) {
	updateBindClaim := UpdateBindClaim{}
	err := json.Unmarshal([]byte(claim), &updateBindClaim)
	if err != nil {
		return UpdateBindClaim{}, ErrInvalidClaim(err.Error())
	}
	return updateBindClaim, nil
}

type SkipSequenceClaim struct {
	ClaimType sdk.ClaimType `json:"claim_type"`
	Sequence  int64         `json:"sequence"`
}

func GetSkipSequenceClaimFromOracleClaim(claim string) (SkipSequenceClaim, sdk.Error) {
	skipSequenceClaim := SkipSequenceClaim{}
	err := json.Unmarshal([]byte(claim), &skipSequenceClaim)
	if err != nil {
		return SkipSequenceClaim{}, ErrInvalidClaim(err.Error())
	}
	return skipSequenceClaim, nil
}
