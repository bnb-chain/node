package timelock

import (
	"bytes"
	"encoding/hex"
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
	accKeyStr := key[:sdk.AddrLen*2]
	accKeyBytes, err := hex.DecodeString(string(accKeyStr))
	if err != nil {
		return []byte{}, 0, err
	}
	addr := sdk.AccAddress(accKeyBytes)

	id, err := strconv.ParseInt(string(key[sdk.AddrLen*2+1:]), 10, 64)
	if err != nil {
		return []byte{}, 0, err
	}
	return addr, id, nil
}
