package types

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ClaimTypeSkipSequence      sdk.ClaimType = 0x1
	ClaimTypeUpdateBind        sdk.ClaimType = 0x2
	ClaimTypeUpdateTransferOut sdk.ClaimType = 0x3
	ClaimTypeTransferIn        sdk.ClaimType = 0x4

	ClaimTypeSkipSequenceName      = "SkipSequence"
	ClaimTypeUpdateBindName        = "UpdateBind"
	ClaimTypeUpdateTransferOutName = "UpdateTransferOut"
	ClaimTypeTransferInName        = "TransferIn"
)

type TransferInClaim struct {
	ContractAddress   EthereumAddress   `json:"contract_address"`
	RefundAddresses   []EthereumAddress `json:"refund_addresses"`
	ReceiverAddresses []sdk.AccAddress  `json:"receiver_addresses"`
	Amounts           []int64           `json:"amounts"`
	Symbol            string            `json:"symbol"`
	RelayFee          sdk.Coin          `json:"relay_fee"`
	ExpireTime        int64             `json:"expire_time"`
}

func GetTransferInClaimFromOracleClaim(claim string) (TransferInClaim, sdk.Error) {
	transferClaim := TransferInClaim{}
	err := json.Unmarshal([]byte(claim), &transferClaim)
	if err != nil {
		return TransferInClaim{}, ErrInvalidTransferMsg(err.Error())
	}
	return transferClaim, nil
}

type UpdateTransferOutClaim struct {
	RefundAddress sdk.AccAddress `json:"refund_address"`
	Amount        sdk.Coin       `json:"amount"`
	RefundReason  RefundReason   `json:"refund_reason"`
}

func GetUpdateTransferOutClaimFromOracleClaim(claim string) (UpdateTransferOutClaim, sdk.Error) {
	updateTransferOutClaim := UpdateTransferOutClaim{}
	err := json.Unmarshal([]byte(claim), &updateTransferOutClaim)
	if err != nil {
		return UpdateTransferOutClaim{}, ErrInvalidTransferMsg(err.Error())
	}
	return updateTransferOutClaim, nil
}

type UpdateBindClaim struct {
	Status           BindStatus      `json:"status"`
	Symbol           string          `json:"symbol"`
	Amount           sdk.Int         `json:"amount"`
	ContractAddress  EthereumAddress `json:"contract_address"`
	ContractDecimals int8            `json:"contract_decimals"`
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
