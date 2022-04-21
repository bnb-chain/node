// Package hd provides basic functionality Hierarchical Deterministic Wallets.
//
// The user must understand the overall concept of the BIP 32 and the BIP 44 specs:
//  https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki
//  https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
//
// In combination with the bip39 package in go-crypto this package provides the functionality for deriving keys using a
// BIP 44 HD path, or, more general, by passing a BIP 32 path.
//
// In particular, this package (together with bip39) provides all necessary functionality to derive keys from
// mnemonics generated during the cosmos fundraiser.
package hd

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcec"
)

// BIP44Prefix is the parts of the BIP32 HD path that are fixed by what we used during the fundraiser.
const (
	BIP44Prefix        = "m/44'/714'/"
	FullFundraiserPath = BIP44Prefix + "0'/0/0"
)

// BIP44Params wraps BIP 44 params (5 level BIP 32 path).
// To receive a canonical string representation ala
// m / purpose' / coin_type' / account' / change / address_index
// call String() on a BIP44Params instance.
type BIP44Params struct {
	purpose    uint32
	coinType   uint32
	account    uint32
	change     bool
	addressIdx uint32
}

// NewParams creates a BIP 44 parameter object from the params:
// m / purpose' / coin_type' / account' / change / address_index
func NewParams(purpose, coinType, account uint32, change bool, addressIdx uint32) *BIP44Params {
	return &BIP44Params{
		purpose:    purpose,
		coinType:   coinType,
		account:    account,
		change:     change,
		addressIdx: addressIdx,
	}
}

// NewParamsFromPath parses the BIP44 path and unmarshals it into a Bip44Params. It supports both absolute and relative paths.
// nolint: gocyclo
func NewParamsFromPath(path string) (*BIP44Params, error) {
	spl := strings.Split(path, "/")

	// Handle absolute or relative paths
	switch {
	case spl[0] == path:
		return nil, fmt.Errorf("path %s doesn't contain '/' separators", path)

	case strings.TrimSpace(spl[0]) == "":
		return nil, fmt.Errorf("ambiguous path %s: use 'm/' prefix for absolute paths, or no leading '/' for relative ones", path)

	case strings.TrimSpace(spl[0]) == "m":
		spl = spl[1:]
	}

	if len(spl) != 5 {
		return nil, fmt.Errorf("invalid path length %s", path)
	}

	if spl[0] != "44'" {
		return nil, fmt.Errorf("first field in path must be 44', got %s", spl[0])
	}

	if !isHardened(spl[1]) || !isHardened(spl[2]) {
		return nil,
			fmt.Errorf("second and third field in path must be hardened (ie. contain the suffix ', got %s and %s", spl[1], spl[2])
	}
	if isHardened(spl[3]) || isHardened(spl[4]) {
		return nil,
			fmt.Errorf("fourth and fifth field in path must not be hardened (ie. not contain the suffix ', got %s and %s", spl[3], spl[4])
	}

	purpose, err := hardenedInt(spl[0])
	if err != nil {
		return nil, fmt.Errorf("invalid HD path purpose %s: %w", spl[0], err)
	}
	coinType, err := hardenedInt(spl[1])
	if err != nil {
		return nil, fmt.Errorf("invalid HD path coin type %s: %w", spl[1], err)
	}
	account, err := hardenedInt(spl[2])
	if err != nil {
		return nil, fmt.Errorf("invalid HD path account %s: %w", spl[2], err)
	}
	change, err := hardenedInt(spl[3])
	if err != nil {
		return nil, fmt.Errorf("invalid HD path change %s: %w", spl[3], err)
	}
	if !(change == 0 || change == 1) {
		return nil, fmt.Errorf("change field can only be 0 or 1")
	}

	addressIdx, err := hardenedInt(spl[4])
	if err != nil {
		return nil, fmt.Errorf("invalid HD path address index %s: %w", spl[4], err)
	}

	return &BIP44Params{
		purpose:    purpose,
		coinType:   coinType,
		account:    account,
		change:     change > 0,
		addressIdx: addressIdx,
	}, nil
}

func hardenedInt(field string) (uint32, error) {
	field = strings.TrimSuffix(field, "'")
	i, err := strconv.Atoi(field)
	if err != nil {
		return 0, err
	}
	if i < 0 {
		return 0, fmt.Errorf("fields must not be negative. got %d", i)
	}
	return uint32(i), nil
}

func isHardened(field string) bool {
	return strings.HasSuffix(field, "'")
}

// NewFundraiserParams creates a BIP 44 parameter object from the params:
// m / 44' / 714' / account' / 0 / address_index
// The fixed parameters (purpose', coin_type', and change) are determined by what was used in the fundraiser.
func NewFundraiserParams(account uint32, addressIdx uint32) *BIP44Params {
	return NewParams(44, 714, account, false, addressIdx)
}

// Return the BIP44 fields as an array.
func (p BIP44Params) DerivationPath() []uint32 {
	change := uint32(0)
	if p.change {
		change = 1
	}
	return []uint32{
		p.purpose,
		p.coinType,
		p.account,
		change,
		p.addressIdx,
	}
}

// String returns the full absolute HD path of the BIP44 (https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki) params:
// m / purpose' / coin_type' / account' / change / address_index
func (p BIP44Params) String() string {
	var changeStr string
	if p.change {
		changeStr = "1"
	} else {
		changeStr = "0"
	}
	return fmt.Sprintf("m/%d'/%d'/%d'/%s/%d",
		p.purpose,
		p.coinType,
		p.account,
		changeStr,
		p.addressIdx)
}

// ComputeMastersFromSeed returns the master public key, master secret, and chain code in hex.
func ComputeMastersFromSeed(seed []byte) (secret [32]byte, chainCode [32]byte) {
	masterSecret := []byte("Bitcoin seed")
	secret, chainCode = i64(masterSecret, seed)

	return
}

// DerivePrivateKeyForPath derives the private key by following the BIP 32/44 path from privKeyBytes,
// using the given chainCode.
func DerivePrivateKeyForPath(privKeyBytes [32]byte, chainCode [32]byte, path string) ([32]byte, error) {
	path = strings.TrimRightFunc(path, func(r rune) bool { return r == filepath.Separator })

	data := privKeyBytes
	parts := strings.Split(path, "/")

	switch {
	case parts[0] == path:
		return [32]byte{}, fmt.Errorf("path '%s' doesn't contain '/' separators", path)
	case strings.TrimSpace(parts[0]) == "m":
		parts = parts[1:]
	}

	for i, part := range parts {
		if part == "" {
			return [32]byte{}, fmt.Errorf("path %q with split element #%d is an empty string", part, i)
		}

		// do we have an apostrophe?
		harden := part[len(part)-1:] == "'"
		// harden == private derivation, else public derivation:
		if harden {
			part = part[:len(part)-1]
		}

		// As per the extended keys specification in
		// https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki#extended-keys
		// index values are in the range [0, 1<<31-1] aka [0, max(int32)]
		idx, err := strconv.ParseUint(part, 10, 31)
		if err != nil {
			return [32]byte{}, fmt.Errorf("invalid BIP 32 path %s: %w", path, err)
		}
		if idx < 0 {
			return [32]byte{}, errors.New("invalid BIP 32 path: index negative or too large")
		}
		data, chainCode = derivePrivateKey(data, chainCode, uint32(idx), harden)
	}
	var derivedKey [32]byte
	n := copy(derivedKey[:], data[:])
	if n != 32 || len(data) != 32 {
		return [32]byte{}, fmt.Errorf("expected a key of length 32, got length: %d", len(data))
	}

	return derivedKey, nil
}

// derivePrivateKey derives the private key with index and chainCode.
// If harden is true, the derivation is 'hardened'.
// It returns the new private key and new chain code.
// For more information on hardened keys see:
//  - https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
func derivePrivateKey(privKeyBytes [32]byte, chainCode [32]byte, index uint32, harden bool) ([32]byte, [32]byte) {
	var data []byte
	if harden {
		index = index | 0x80000000
		data = append([]byte{byte(0)}, privKeyBytes[:]...)
	} else {
		// this can't return an error:
		_, ecPub := btcec.PrivKeyFromBytes(btcec.S256(), privKeyBytes[:])
		pubkeyBytes := ecPub.SerializeCompressed()
		data = pubkeyBytes

		/* By using btcec, we can remove the dependency on tendermint/crypto/secp256k1
		pubkey := secp256k1.PrivKeySecp256k1(privKeyBytes).PubKey()
		public := pubkey.(secp256k1.PubKeySecp256k1)
		data = public[:]
		*/
	}
	data = append(data, uint32ToBytes(index)...)
	data2, chainCode2 := i64(chainCode[:], data)
	x := addScalars(privKeyBytes[:], data2[:])
	return x, chainCode2
}

// modular big endian addition
func addScalars(a []byte, b []byte) [32]byte {
	aInt := new(big.Int).SetBytes(a)
	bInt := new(big.Int).SetBytes(b)
	sInt := new(big.Int).Add(aInt, bInt)
	x := sInt.Mod(sInt, btcec.S256().N).Bytes()
	x2 := [32]byte{}
	copy(x2[32-len(x):], x)
	return x2
}

func uint32ToBytes(i uint32) []byte {
	b := [4]byte{}
	binary.BigEndian.PutUint32(b[:], i)
	return b[:]
}

// i64 returns the two halfs of the SHA512 HMAC of key and data.
func i64(key []byte, data []byte) (IL [32]byte, IR [32]byte) {
	mac := hmac.New(sha512.New, key)
	// sha512 does not err
	_, _ = mac.Write(data)
	I := mac.Sum(nil)
	copy(IL[:], I[:32])
	copy(IR[:], I[32:])
	return
}
