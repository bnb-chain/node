package types

import (
	"encoding/json"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/plugins/oracle"
)

const (
	ClaimIdDelimiter = "-"

	TransferChannelId           uint8 = 1
	TransferOutTimeoutChannelId uint8 = 2
	UpdateBindChannelId         uint8 = 3
)

func GetClaimId(channel uint8, sequence int64) string {
	return strconv.FormatInt(int64(channel), 10) + ClaimIdDelimiter + strconv.FormatInt(sequence, 10)
}

type TransferClaim struct {
	ContractAddress EthereumAddress `json:"contract_address"`
	SenderAddress   EthereumAddress `json:"sender_address"`
	ReceiverAddress sdk.AccAddress  `json:"receiver_address"`
	Amount          sdk.Coin        `json:"amount"`
	RelayFee        sdk.Coin        `json:"relay_fee"`
	ExpireTime      int64           `json:"expire_time"`
}

func CreateOracleClaimFromTransferMsg(msg TransferInMsg) (oracle.Claim, sdk.Error) {
	claimId := GetClaimId(TransferChannelId, msg.Sequence)
	transferClaim := TransferClaim{
		ContractAddress: msg.ContractAddress,
		SenderAddress:   msg.SenderAddress,
		ReceiverAddress: msg.ReceiverAddress,
		Amount:          msg.Amount,
		RelayFee:        msg.RelayFee,
		ExpireTime:      msg.ExpireTime,
	}
	claimBytes, err := json.Marshal(transferClaim)
	if err != nil {
		return oracle.Claim{}, ErrInvalidTransferMsg(err.Error())
	}
	claim := oracle.NewClaim(claimId, sdk.ValAddress(msg.ValidatorAddress), string(claimBytes))
	return claim, nil
}

func GetTransferClaimFromOracleClaim(claim string) (TransferClaim, sdk.Error) {
	transferClaim := TransferClaim{}
	err := json.Unmarshal([]byte(claim), &transferClaim)
	if err != nil {
		return TransferClaim{}, ErrInvalidTransferMsg(err.Error())
	}
	return transferClaim, nil
}

type TransferOutTimeoutClaim struct {
	SenderAddress sdk.AccAddress `json:"sender_address"`
	Amount        sdk.Coin       `json:"amount"`
}

func CreateOracleClaimFromTransferOutTimeoutMsg(msg TransferOutTimeoutMsg) (oracle.Claim, sdk.Error) {
	claimId := GetClaimId(TransferOutTimeoutChannelId, msg.Sequence)

	transferOutTimeoutClaim := TransferOutTimeoutClaim{
		SenderAddress: msg.SenderAddress,
		Amount:        msg.Amount,
	}
	claimBytes, err := json.Marshal(transferOutTimeoutClaim)
	if err != nil {
		return oracle.Claim{}, ErrInvalidTransferMsg(err.Error())
	}
	claim := oracle.NewClaim(claimId, sdk.ValAddress(msg.ValidatorAddress), string(claimBytes))
	return claim, nil
}

func GetTransferOutTimeoutClaimFromOracleClaim(claim string) (TransferOutTimeoutClaim, sdk.Error) {
	transferOutTimeoutClaim := TransferOutTimeoutClaim{}
	err := json.Unmarshal([]byte(claim), &transferOutTimeoutClaim)
	if err != nil {
		return TransferOutTimeoutClaim{}, ErrInvalidTransferMsg(err.Error())
	}
	return transferOutTimeoutClaim, nil
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
