package timelock

import (
	"bytes"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func KeyRecord(addr sdk.AccAddress, id int64) []byte {
	return []byte(fmt.Sprintf("record:%d:%d", addr, id))
}

func KeyRecordSubSpace(addr sdk.AccAddress) []byte {
	return []byte(fmt.Sprintf("record:%d", addr))
}

func ParseKeyRecord(key []byte) (sdk.AccAddress, int64, error) {
	key = bytes.TrimPrefix(key, []byte("record:"))
	addr := sdk.AccAddress(key[:sdk.AddrLen])

	id, err := strconv.ParseInt(string(key[sdk.AddrLen+1:]), 10, 64)
	if err != nil {
		return []byte{}, 0, err
	}
	return addr, id, nil
}
