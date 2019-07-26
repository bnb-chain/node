package swap

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	Open      = iota
	Completed = iota
	Expired   = iota

	RandomNumberHashLength = 32
	RandomNumberLength     = 32
)

type AtomicSwap struct {
	From      sdk.AccAddress `json:"from"`
	To        sdk.AccAddress `json:"to"`
	OutAmount sdk.Coin       `json:"out_amount"`

	InAmount       uint64 `json:"in_amount"`
	ToOnOtherChain []byte `json:"to_on_other_chain"`

	RandomNumberHash []byte `json:"random_number_hash"`
	RandomNumber     []byte `json:"random_number"`
	Timestamp        uint64 `json:"timestamp"`

	ExpireHeight int64     `json:"expire_height"`
	ClosedTime   int64     `json:"closed_time"`
	Status       int8      `json:"status"`
}
