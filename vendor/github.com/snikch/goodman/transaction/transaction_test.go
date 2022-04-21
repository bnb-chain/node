package transaction

import "testing"

func TestSettingFailToString(t *testing.T) {
	trans := Transaction{}
	trans.Fail = "Fail"

	if trans.Fail != "Fail" {
		t.Errorf("Transaction.Fail member must be able to be set as string")
	}
}
