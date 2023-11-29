package timelock

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestParseKeyRecord(t *testing.T) {
	account, err := sdk.AccAddressFromHex("5B38Da6a701c568545dCfcB03FcB875f56beddC4")
	if err != nil {
		t.Fatal(err)
		return
	}
	accId := int64(1513)
	key := KeyRecord(account, accId)

	acc, id, err := ParseKeyRecord([]byte(key))
	if err != nil {
		t.Fatal(err)
		return
	}

	if !acc.Equals(account) {
		t.Fatal("parse account error")
		return
	}
	if id != accId {
		t.Fatal("parse id error")
		return
	}
}
