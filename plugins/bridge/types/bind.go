package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type BindRequest struct {
	From             sdk.AccAddress    `json:"from"`
	Symbol           string            `json:"symbol"`
	Amount           sdk.Int           `json:"amount"`
	ContractAddress  SmartChainAddress `json:"contract_address"`
	ContractDecimals int8              `json:"contract_decimals"`
	ExpireTime       int64             `json:"expire_time"`
}
