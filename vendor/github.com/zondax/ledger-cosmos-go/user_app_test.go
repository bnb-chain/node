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

package ledger_cosmos_go

import (
	"encoding/hex"
	"fmt"
	secp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"strings"
	"testing"
)

// Ledger Test Mnemonic: equip will roof matter pink blind book anxiety banner elbow sun young

func Test_UserFindLedger(t *testing.T) {
	userApp, err := FindLedgerCosmosUserApp()
	if err != nil {
		t.Fatalf(err.Error())
	}

	assert.NotNil(t, userApp)
	defer userApp.Close()
}

func Test_UserGetVersion(t *testing.T) {
	userApp, err := FindLedgerCosmosUserApp()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer userApp.Close()

	userApp.api.Logging = true

	version, err := userApp.GetVersion()
	require.Nil(t, err, "Detected error")
	fmt.Println(version)

	assert.Equal(t, uint8(0x0), version.AppMode, "TESTING MODE ENABLED!!")
	assert.Equal(t, uint8(0x1), version.Major, "Wrong Major version")
	assert.Equal(t, uint8(0x1), version.Minor, "Wrong Minor version")
	assert.Equal(t, uint8(0x4), version.Patch, "Wrong Patch version")
}

func Test_UserGetPublicKey(t *testing.T) {
	userApp, err := FindLedgerCosmosUserApp()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer userApp.Close()

	userApp.api.Logging = true

	path := []uint32{44, 714, 0, 0, 0}

	pubKey, err := userApp.GetPublicKeySECP256K1(path)
	if err != nil {
		t.Fatalf("Detected error, err: %s\n", err.Error())
	}

	assert.Equal(
		t,
		65,
		len(pubKey),
		"Public key has wrong length: %x, expected length: %x\n", pubKey, 65)

	fmt.Printf("PUBLIC KEY: %x\n", pubKey)

	_, err = secp256k1.ParsePubKey(pubKey[:], secp256k1.S256())
	require.Nil(t, err, "Error parsing public key err: %s\n", err)
}

func Test_UserShowAddresses(t *testing.T) {
	userApp, err := FindLedgerCosmosUserApp()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer userApp.Close()

	userApp.api.Logging = true

	hrp := "bnb"
	path := []uint32{44, 714, 0, 0, 0}

	err = userApp.ShowAddressSECP256K1(path, hrp)
	if err != nil {
		t.Fatalf("Detected error, err: %s\n", err.Error())
	}
}

func Test_UserPK_HDPaths(t *testing.T) {
	userApp, err := FindLedgerCosmosUserApp()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer userApp.Close()

	userApp.api.Logging = true

	path := []uint32{44, 714, 0, 0, 0}

	expected := []string{
		"04f937728d58ec329e41ce1a6bb447880e6b9bc20873c4e00f4a91f9ee095cb688d5d1f02a31012b6d294d8990d147c8b364e91b706052fe33b57641c9b64c9053",
		"04a038044f970057da9a922571cd6b1f67810e793b9e6b7396309e31b6ebf35d2327c141f980ae898d5d39f4231d5004861583a304b43a129bb1512965bbe708d2",
		"044d80837bb3f6f7a7868f92e390f5e8bc9a6093bdf6235f70c6d9d6acfa50c9e476a9964eea590e416b12f48437d4a8731a361d92a841582948141c7e203b6388",
		"04385ee0b85b9fa059731671b5c86f9e0ac1b1a653ba771076082f18d45c94ce69f2907a9e55ae099a2018264abbe940024df09993383274ab640da938a82b76fd",
		"0462adbcccd8f6720d1d844bd81067bd8672460539015a33ed8f932a3f750dfd77526897515247561bc0f3e740e3cf02e344f2baa76579e26e9bc0232dc4d964c1",
		"047f6cef691c1a0361f299e744d457b8555b7e06243b9ab657e15f79f7a4bc6560fc5b946837377c258c0b1e4e3d0601a5056150f1de538da667a088efede1a4af",
		"04021bea9dcd633288859e3158fae431939aa8bfee59632a9a745328ecd81091ee23614c2390b177b5d9523599d3bc5d34c731e830d9da4bc6cdd4a1fca47c8c36",
		"04a25204a24a5f70e14b67fb1e0d3a0424b1f136538214ca7a5c0c9ef548b321142e909ac25070e7c583abef4d12645d469d9d1254b001497c0d6d288a90975d78",
		"04483b48b0b6b97f8450352f92cb9336f4d9692a055afdc9ba4180077e1b284af1a2f54976b8c5dce8e9db872419158d93fdcdfa4778a18332c6594c30bb6eb1a1",
		"040eff3c04290f3ccbf46ce1b1a710af9ef9e88858aeaf48f1f36b8fb0cf7d1201c7bacc3ebec4a2a53a8b19203a35948892852492b040262bbfecad6a3fee32d7",
	}

	for i := uint32(0); i < 10; i++ {
		path[4] = i

		pubKey, err := userApp.GetPublicKeySECP256K1(path)
		if err != nil {
			t.Fatalf("Detected error, err: %s\n", err.Error())
		}

		assert.Equal(
			t,
			65,
			len(pubKey),
			"Public key has wrong length: %x, expected length: %x\n", pubKey, 65)

		assert.Equal(
			t,
			expected[i],
			hex.EncodeToString(pubKey),
			"Public key 44'/714'/0'/0/%d does not match\n", i)

		_, err = secp256k1.ParsePubKey(pubKey[:], secp256k1.S256())
		require.Nil(t, err, "Error parsing public key err: %s\n", err)

	}
}

func getDummyTx() []byte {
	dummyTx := `{
		"account_number": 1,
		"chain_id": "some_chain",
		"data": "data",
		"memo": "MEMO",
		"msgs": ["SOMETHING"],
		"sequence": 3,
		"source": 0
	}`
	dummyTx = strings.Replace(dummyTx, " ", "", -1)
	dummyTx = strings.Replace(dummyTx, "\n", "", -1)
	dummyTx = strings.Replace(dummyTx, "\t", "", -1)

	return []byte(dummyTx)
}

func Test_UserSign(t *testing.T) {
	userApp, err := FindLedgerCosmosUserApp()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer userApp.Close()

	userApp.api.Logging = true

	path := []uint32{44, 714, 0, 0, 0}

	message := getDummyTx()
	signature, err := userApp.SignSECP256K1(path, message)
	if err != nil {
		t.Fatalf("[Sign] Error: %s\n", err.Error())
	}

	// Verify Signature
	pubKey, err := userApp.GetPublicKeySECP256K1(path)
	if err != nil {
		t.Fatalf("Detected error, err: %s\n", err.Error())
	}

	if err != nil {
		t.Fatalf("[GetPK] Error: " + err.Error())
		return
	}

	pub2, err := secp256k1.ParsePubKey(pubKey[:], secp256k1.S256())
	if err != nil {
		t.Fatalf("[ParsePK] Error: " + err.Error())
		return
	}

	sig2, err := secp256k1.ParseDERSignature(signature[:], secp256k1.S256())
	if err != nil {
		t.Fatalf("[ParseSig] Error: " + err.Error())
		return
	}

	verified := sig2.Verify(crypto.Sha256(message), pub2)
	if !verified {
		t.Fatalf("[VerifySig] Error verifying signature: " + err.Error())
		return
	}
}

func Test_UserSign_Fails(t *testing.T) {
	userApp, err := FindLedgerCosmosUserApp()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer userApp.Close()

	userApp.api.Logging = true

	path := []uint32{44, 714, 0, 0, 5}

	message := getDummyTx()
	garbage := []byte{65}
	message = append(garbage, message...)

	_, err = userApp.SignSECP256K1(path, message)
	assert.EqualError(t, err, "[APDU_CODE_DATA_INVALID] Referenced data reversibly blocked (invalidated)")
}
