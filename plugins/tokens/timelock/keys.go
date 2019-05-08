package timelock

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func KeyRecord(addr sdk.AccAddress, id int64) []byte {
	return []byte(fmt.Sprintf("record:%d:%d", addr, id))
}

func KeyRecordSubSpace(addr sdk.AccAddress) []byte {
	return []byte(fmt.Sprintf("record:%d", addr))
}

func KeyNextRecordId(addr sdk.AccAddress) []byte {
	return []byte(fmt.Sprintf("newRecordId:%d", addr))
}
