package hd

import (
	"encoding/hex"
	"fmt"
	"testing"

	bip39 "github.com/cosmos/go-bip39"
	"github.com/stretchr/testify/require"
)

var defaultBIP39Passphrase = ""

// return bip39 seed with empty passphrase
func mnemonicToSeed(mnemonic string) []byte {
	return bip39.NewSeed(mnemonic, defaultBIP39Passphrase)
}

func TestPathParamsString(t *testing.T) {
	path := NewParams(44, 0, 0, false, 0)
	require.Equal(t, "m/44'/0'/0'/0/0", path.String())
	path = NewParams(44, 33, 7, true, 9)
	require.Equal(t, "m/44'/33'/7'/1/9", path.String())
}

func TestStringifyFundraiserPathParams(t *testing.T) {
	path := NewFundraiserParams(4,  22)
	require.Equal(t, "m/44'/714'/4'/0/22", path.String())

	path = NewFundraiserParams(4, 57)
	require.Equal(t, "m/44'/714'/4'/0/57", path.String())
}

func TestPathToArray(t *testing.T) {
	path := NewParams(44, 118, 1, false, 4)
	require.Equal(t, "[44 118 1 0 4]", fmt.Sprintf("%v", path.DerivationPath()))

	path = NewParams(44, 118, 2, true, 15)
	require.Equal(t, "[44 118 2 1 15]", fmt.Sprintf("%v", path.DerivationPath()))
}

func TestParamsFromPath(t *testing.T) {
	goodCases := []struct {
		params *BIP44Params
		path   string
	}{
		{&BIP44Params{44, 0, 0, false, 0}, "m/44'/0'/0'/0/0"},
		{&BIP44Params{44, 1, 0, false, 0}, "m/44'/1'/0'/0/0"},
		{&BIP44Params{44, 0, 1, false, 0}, "m/44'/0'/1'/0/0"},
		{&BIP44Params{44, 0, 0, true, 0}, "m/44'/0'/0'/1/0"},
		{&BIP44Params{44, 0, 0, false, 1}, "m/44'/0'/0'/0/1"},
		{&BIP44Params{44, 1, 1, true, 1}, "m/44'/1'/1'/1/1"},
		{&BIP44Params{44, 118, 52, true, 41}, "m/44'/118'/52'/1/41"},
	}

	for i, c := range goodCases {
		params, err := NewParamsFromPath(c.path)
		errStr := fmt.Sprintf("%d %v", i, c)
		require.NoError(t, err, errStr)
		require.EqualValues(t, c.params, params, errStr)
		require.Equal(t, c.path, c.params.String())
	}

	badCases := []struct {
		path string
	}{
		{"m/43'/0'/0'/0/0"},   // doesn't start with 44
		{"m/44'/1'/0'/0/0/5"}, // too many fields
		{"m/44'/0'/1'/0"},     // too few fields
		{"m/44'/0'/0'/2/0"},   // change field can only be 0/1
		{"m/44/0'/0'/0/0"},    // first field needs '
		{"m/44'/0/0'/0/0"},    // second field needs '
		{"m/44'/0'/0/0/0"},    // third field needs '
		{"m/44'/0'/0'/0'/0"},  // fourth field must not have '
		{"m/44'/0'/0'/0/0'"},  // fifth field must not have '
		{"m/44'/-1'/0'/0/0"},  // no negatives
		{"m/44'/0'/0'/-1/0"},  // no negatives
		{"m/a'/0'/0'/-1/0"},   // invalid values
		{"m/0/X/0'/-1/0"},     // invalid values
		{"m/44'/0'/X/-1/0"},   // invalid values
		{"m/44'/0'/0'/%/0"},   // invalid values
		{"m/44'/0'/0'/0/%"},   // invalid values
		{"m44'0'0'00"},        // no separators
		{" /44'/0'/0'/0/0"},   // blank first component
	}

	for i, c := range badCases {
		params, err := NewParamsFromPath(c.path)
		errStr := fmt.Sprintf("%d %v", i, c)
		require.Nil(t, params, errStr)
		require.Error(t, err, errStr)
	}

}

func TestBIP32Vecs(t *testing.T) {
	seed := mnemonicToSeed("barrel original fuel morning among eternal " +
		"filter ball stove pluck matrix mechanic")
	master, ch := ComputeMastersFromSeed(seed)
	fmt.Println("keys from fundraiser test-vector (cosmos, bitcoin, ether)")
	fmt.Println()

	// cosmos, absolute path
	priv, err := DerivePrivateKeyForPath(master, ch, FullFundraiserPath)
	require.NoError(t, err)
	require.NotEmpty(t, priv)
	fmt.Println(hex.EncodeToString(priv[:]))

	absPrivKey := hex.EncodeToString(priv[:])

	// cosmos, relative path
	priv, err = DerivePrivateKeyForPath(master, ch, "44'/714'/0'/0/0")
	require.NoError(t, err)
	require.NotEmpty(t, priv)

	relPrivKey := hex.EncodeToString(priv[:])

	// check compatibility between relative and absolute HD paths
	require.Equal(t, relPrivKey, absPrivKey)

	// bitcoin
	priv, err = DerivePrivateKeyForPath(master, ch, "m/44'/0'/0'/0/0")
	require.NoError(t, err)
	require.NotEmpty(t, priv)
	fmt.Println(hex.EncodeToString(priv[:]))

	// ether
	priv, err = DerivePrivateKeyForPath(master, ch, "m/44'/60'/0'/0/0")
	require.NoError(t, err)
	require.NotEmpty(t, priv)
	fmt.Println(hex.EncodeToString(priv[:]))

	// INVALID
	priv, err = DerivePrivateKeyForPath(master, ch, "m/X/0'/0'/0/0")
	require.Error(t, err)

	priv, err = DerivePrivateKeyForPath(master, ch, "m/-44/0'/0'/0/0")
	require.Error(t, err)

	fmt.Println()
	fmt.Println("keys generated via https://coinomi.com/recovery-phrase-tool.html")
	fmt.Println()

	seed = mnemonicToSeed(
		"advice process birth april short trust crater change bacon monkey medal garment " +
			"gorilla ranch hour rival razor call lunar mention taste vacant woman sister")
	master, ch = ComputeMastersFromSeed(seed)
	priv, _ = DerivePrivateKeyForPath(master, ch, "m/44'/1'/1'/0/4")
	fmt.Println(hex.EncodeToString(priv[:]))

	seed = mnemonicToSeed("idea naive region square margin day captain habit " +
		"gun second farm pact pulse someone armed")
	master, ch = ComputeMastersFromSeed(seed)
	priv, err = DerivePrivateKeyForPath(master, ch, "m/44'/0'/0'/0/420")
	require.NoError(t, err)
	fmt.Println(hex.EncodeToString(priv[:]))

	fmt.Println()
	fmt.Println("BIP 32 example")
	fmt.Println()

	// bip32 path: m/0/7
	seed = mnemonicToSeed("monitor flock loyal sick object grunt duty ride develop assault harsh history")
	master, ch = ComputeMastersFromSeed(seed)
	priv, err = DerivePrivateKeyForPath(master, ch, "m/0/7")
	require.NoError(t, err) // TODO: shouldn't this error?
	fmt.Println(hex.EncodeToString(priv[:]))

	// Output: keys from fundraiser test-vector (cosmos, bitcoin, ether)
	//
	// 01dcb36acfd5de52ac1f00daf231e64637388202f1fce7bdc64f6bb3199d270d
	// e77c3de76965ad89997451de97b95bb65ede23a6bf185a55d80363d92ee37c3d
	// 7fc4d8a8146dea344ba04c593517d3f377fa6cded36cd55aee0a0bb968e651bc
	//
	// keys generated via https://coinomi.com/recovery-phrase-tool.html
	//
	// a61f10c5fecf40c084c94fa54273b6f5d7989386be4a37669e6d6f7b0169c163
	// 32c4599843de3ef161a629a461d12c60b009b676c35050be5f7ded3a3b23501f
	//
	// BIP 32 example
	//
	// c4c11d8c03625515905d7e89d25dfc66126fbc629ecca6db489a1a72fc4bda78
}


// Tests to ensure that any index value is in the range [0, max(int32)] as per
// the extended keys specification. If the index belongs to that of a hardened key,
// its 0x80000000 bit will be set, so we can still accept values in [0, max(int32)] and then
// increase its value as deriveKeyPath already augments.
// See issue https://github.com/cosmos/cosmos-sdk/issues/7627.
func TestDeriveHDPathRange(t *testing.T) {
	seed := mnemonicToSeed("I am become Death, the destroyer of worlds!")

	tests := []struct {
		path    string
		wantErr string
	}{
		{
			path:    "m/1'/2147483648/0'/0/0",
			wantErr: "out of range",
		},
		{
			path:    "m/2147483648'/1/0/0",
			wantErr: "out of range",
		},
		{
			path:    "m/2147483648'/2147483648/0'/0/0",
			wantErr: "out of range",
		},
		{
			path:    "m/1'/-5/0'/0/0",
			wantErr: "invalid syntax",
		},
		{
			path:    "m/-2147483646'/1/0/0",
			wantErr: "invalid syntax",
		},
		{
			path:    "m/-2147483648'/-2147483648/0'/0/0",
			wantErr: "invalid syntax",
		},
		{
			path:    "m44'118'0'00",
			wantErr: "path 'm44'118'0'00' doesn't contain '/' separators",
		},
		{
			path:    "",
			wantErr: "path '' doesn't contain '/' separators",
		},
		{
			// Should pass.
			path: "m/1'/2147483647'/1/0'/0/0",
		},
		{
			// Should pass.
			path: "1'/2147483647'/1/0'/0/0",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			master, ch := ComputeMastersFromSeed(seed)
			_, err := DerivePrivateKeyForPath(master, ch, tt.path)

			if tt.wantErr == "" {
				require.NoError(t, err, "unexpected error")
			} else {
				require.Error(t, err, "expected a report of an int overflow")
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// Ensuring that we don't crash if values have trailing slashes
func TestDerivePrivateKeyForPathDoNotCrash(t *testing.T) {
	paths := []string{
		"m/5/",
		"m/5",
		"/44",
		"m//5",
		"m/0/7",
		"/",
		" m       /0/7",
		"              /       ",
		"m///7//////",
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			DerivePrivateKeyForPath([32]byte{}, [32]byte{}, path)
		})
	}
}
