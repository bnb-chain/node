// +build ledger_device

/*******************************************************************************
*   (c) 2018 ZondaX GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
********************************************************************************/

package ledger_go

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/zondax/hid"
	"testing"
)

func Test_ThereAreDevices(t *testing.T) {
	devices := hid.Enumerate(0, 0)
	assert.NotEqual(t, 0, len(devices))
}

func Test_ListDevices(t *testing.T) {
	ListDevices()
}

func Test_FindLedger(t *testing.T) {
	ledger, err := FindLedger()
	if err != nil {
		fmt.Println("\n*********************************")
		fmt.Println("Did you enter the password??")
		fmt.Println("*********************************")
		t.Fatalf("Error: %s", err.Error())
	}
	assert.NotNil(t, ledger)
}

func Test_BasicExchange(t *testing.T) {
	ledger, err := FindLedger()
	if err != nil {
		fmt.Println("\n*********************************")
		fmt.Println("Did you enter the password??")
		fmt.Println("*********************************")
		t.Fatalf("Error: %s", err.Error())
	}
	assert.NotNil(t, ledger)

	message := []byte{0x55, 0, 0, 0, 0}

	for i := 0; i < 10; i++ {
		response, err := ledger.Exchange(message)

		if err != nil {
			fmt.Printf("iteration %d\n", i)
			t.Fatalf("Error: %s", err.Error())
		}

		assert.Equal(t, 5, len(response))
	}
}

func Test_LongExchange(t *testing.T) {
	ledger, err := FindLedger()
	if err != nil {
		fmt.Println("\n*********************************")
		fmt.Println("Did you enter the password??")
		fmt.Println("*********************************")
		t.Fatalf("Error: %s", err.Error())
	}
	assert.NotNil(t, ledger)

	path := "052c000080760000800000008000000000000000000000000000000000000000000000000000000000"
	pathBytes, err := hex.DecodeString(path)
	if err != nil {
		t.Fatalf("invalid path in test")
	}

	header := []byte{0x55, 1, 0, 0, byte(len(pathBytes))}
	message := append(header, pathBytes...)

	response, err := ledger.Exchange(message)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	assert.Equal(t, 65, len(response))
}
