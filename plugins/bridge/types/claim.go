package types

import (
	"encoding/json"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/plugins/oracle"
)

const (
	ClaimIdDelimiter = "-"

	TransferInChannelId        uint8 = 1
	UpdateTransferOutChannelId uint8 = 2
	UpdateBindChannelId        uint8 = 3
)

func GetClaimId(channel uint8, sequence int64) string {
	return strconv.FormatInt(int64(channel), 10) + ClaimIdDelimiter + strconv.FormatInt(sequence, 10)
}

type TransferInClaim struct {
	ContractAddress   EthereumAddress   `json:"contract_address"`
	RefundAddresses   []EthereumAddress `json:"refund_addresses"`
	ReceiverAddresses []sdk.AccAddress  `json:"receiver_addresses"`
	Amounts           []int64           `json:"amounts"`
	Symbol            string            `json:"symbol"`
	RelayFee          sdk.Coin          `json:"relay_fee"`
	ExpireTime        int64             `json:"expire_time"`
}

func CreateOracleClaimFromTransferInMsg(msg TransferInMsg) (oracle.Claim, sdk.Error) {
	claimId := GetClaimId(TransferInChannelId, msg.Sequence)
	transferClaim := TransferInClaim{
		ContractAddress:   msg.ContractAddress,
		RefundAddresses:   msg.RefundAddresses,
		ReceiverAddresses: msg.ReceiverAddresses,
		Amounts:           msg.Amounts,
		Symbol:            strings.ToUpper(msg.Symbol),
		RelayFee:          msg.RelayFee,
		ExpireTime:        msg.ExpireTime,
	}
	claimBytes, err := json.Marshal(transferClaim)
	if err != nil {
		return oracle.Claim{}, ErrInvalidTransferMsg(err.Error())
	}
	claim := oracle.NewClaim(claimId, sdk.ValAddress(msg.ValidatorAddress), string(claimBytes))
	return claim, nil
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

func CreateOracleClaimFromUpdateTransferOutMsg(msg UpdateTransferOutMsg) (oracle.Claim, sdk.Error) {
	claimId := GetClaimId(UpdateTransferOutChannelId, msg.Sequence)

	updateTransferOutClaim := UpdateTransferOutClaim{
		RefundAddress: msg.RefundAddress,
		Amount:        msg.Amount,
		RefundReason:  msg.RefundReason,
	}
	claimBytes, err := json.Marshal(updateTransferOutClaim)
	if err != nil {
		return oracle.Claim{}, ErrInvalidTransferMsg(err.Error())
	}
	claim := oracle.NewClaim(claimId, sdk.ValAddress(msg.ValidatorAddress), string(claimBytes))
	return claim, nil
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
	Amount           int64           `json:"amount"`
	ContractAddress  EthereumAddress `json:"contract_address"`
	ContractDecimals int8            `json:"contract_decimals"`
}

func CreateOracleClaimFromUpdateBindMsg(msg UpdateBindMsg) (oracle.Claim, sdk.Error) {
	claimId := GetClaimId(UpdateBindChannelId, msg.Sequence)

	updateBindClaim := UpdateBindMsg{
		Status:           msg.Status,
		Symbol:           msg.Symbol,
		Amount:           msg.Amount,
		ContractAddress:  msg.ContractAddress,
		ContractDecimals: msg.ContractDecimals,
	}
	claimBytes, err := json.Marshal(updateBindClaim)
	if err != nil {
		return oracle.Claim{}, ErrInvalidTransferMsg(err.Error())
	}
	claim := oracle.NewClaim(claimId, sdk.ValAddress(msg.ValidatorAddress), string(claimBytes))
	return claim, nil
}

func GetUpdateBindClaimFromOracleClaim(claim string) (UpdateBindMsg, sdk.Error) {
	updateBindClaim := UpdateBindMsg{}
	err := json.Unmarshal([]byte(claim), &updateBindClaim)
	if err != nil {
		return UpdateBindMsg{}, ErrInvalidTransferMsg(err.Error())
	}
	return updateBindClaim, nil
}
