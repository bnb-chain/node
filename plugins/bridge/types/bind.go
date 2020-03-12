package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type BindRequest struct {
	From            sdk.AccAddress  `json:"from"`
	Symbol          string          `json:"symbol"`
	Amount          int64           `json:"amount"`
	ContractAddress EthereumAddress `json:"contract_address"`
	ContractDecimal int8            `json:"contract_decimal"`
	ExpireTime      int64           `json:"expire_time"`
}

func GetBindRequest(msg BindMsg) BindRequest {
	return BindRequest{
		From:            msg.From,
		Symbol:          msg.Symbol,
		Amount:          msg.Amount,
		ContractAddress: msg.ContractAddress,
		ContractDecimal: msg.ContractDecimal,
		ExpireTime:      msg.ExpireTime,
	}
}
