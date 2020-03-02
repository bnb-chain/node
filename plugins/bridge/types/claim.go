package types

import (
	"encoding/json"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/plugins/oracle"
)

const (
	ClaimIdDelimiter = "-"

	TransferChannelId uint8 = 1
	TimeoutChannelId  uint8 = 2
)

func GetClaimId(channel uint8, sequence int64) string {
	return strconv.FormatInt(int64(channel), 10) + ClaimIdDelimiter + strconv.FormatInt(sequence, 10)
}

type TransferClaim struct {
	ReceiverAddress sdk.AccAddress `json:"receiver_address"`
	Amount          sdk.Coin       `json:"amount"`
	RelayFee        sdk.Coin       `json:"relay_fee"`
}

func CreateOracleClaimFromTransferMsg(msg TransferMsg) (oracle.Claim, sdk.Error) {
	claimId := GetClaimId(TransferChannelId, msg.Sequence)
	transferClaim := TransferClaim{
		ReceiverAddress: msg.ReceiverAddress,
		Amount:          msg.Amount,
		RelayFee:        msg.RelayFee,
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

type TimeoutClaim struct {
	SenderAddress sdk.AccAddress `json:"sender_address"`
	Amount        sdk.Coin       `json:"amount"`
}

func CreateOracleClaimFromTimeoutMsg(msg TimeoutMsg) (oracle.Claim, sdk.Error) {
	claimId := GetClaimId(TimeoutChannelId, msg.Sequence)

	timeoutClaim := TimeoutClaim{
		SenderAddress: msg.SenderAddress,
		Amount:        msg.Amount,
	}
	claimBytes, err := json.Marshal(timeoutClaim)
	if err != nil {
		return oracle.Claim{}, ErrInvalidTransferMsg(err.Error())
	}
	claim := oracle.NewClaim(claimId, sdk.ValAddress(msg.ValidatorAddress), string(claimBytes))
	return claim, nil
}

func GetTimeoutClaimFromOracleClaim(claim string) (TimeoutClaim, sdk.Error) {
	timeoutClaim := TimeoutClaim{}
	err := json.Unmarshal([]byte(claim), &timeoutClaim)
	if err != nil {
		return TimeoutClaim{}, ErrInvalidTransferMsg(err.Error())
	}
	return timeoutClaim, nil
}
