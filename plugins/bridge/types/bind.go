package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type BindRequest struct {
	From             sdk.AccAddress        `json:"from"`
	Symbol           string                `json:"symbol"`
	Amount           int64                 `json:"amount"`
	DeductedAmount   int64                 `json:"deducted_amount"`
	ContractAddress  sdk.SmartChainAddress `json:"contract_address"`
	ContractDecimals int8                  `json:"contract_decimals"`
	ExpireTime       int64                 `json:"expire_time"`
}
