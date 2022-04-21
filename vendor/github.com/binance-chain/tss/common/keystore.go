package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"path"

	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/paillier"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/sha3"
)

const (
	cipherAlg = "aes-256-ctr"

	// This is essentially a hybrid of the Argon2d and Argon2i algorithms and uses a combination of
	// data-independent memory access (for resistance against side-channel timing attacks) and
	// data-depending memory access (for resistance against GPU cracking attacks).
	keyHeaderKDF = "Argon2id"
)

type cryptoJSON struct {
	Cipher       string           `json:"cipher"`
	CipherText   string           `json:"ciphertext"`
	CipherParams cipherparamsJSON `json:"cipherparams"`
	KDF          string           `json:"kdf"`
	KDFParams    KDFConfig        `json:"kdfparams"`
	MAC          string           `json:"mac"`
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

// derived from keygen.LocalPartySaveData
type secretFields struct {
	Xi         *big.Int             // xi, kj
	PaillierSk *paillier.PrivateKey // ski
	NodeKey    []byte
}

// derived from keygen.LocalPartySaveData
type publicFields struct {
	ShareID           *big.Int
	BigXj             []*crypto.ECPoint     // Xj
	ECDSAPub          *crypto.ECPoint       // y
	PaillierPks       []*paillier.PublicKey // pkj
	NTildej, H1j, H2j []*big.Int
	NTildei, H1i, H2i *big.Int
	Ks                []*big.Int
}

// crypto.ECPoint is not json marshallable
func (data *publicFields) MarshalJSON() ([]byte, error) {
	bigXj, err := crypto.FlattenECPoints(data.BigXj)
	if err != nil {
		return nil, errors.New("failed to flatten bigXjs")
	}
	ecdsaPub, err := crypto.FlattenECPoints([]*crypto.ECPoint{data.ECDSAPub})
	if err != nil {
		return nil, errors.New("failed to flatten ecdsa public key")
	}

	type Alias publicFields
	return json.Marshal(&struct {
		BigXj    []*big.Int
		ECDSAPub []*big.Int
		*Alias
	}{
		BigXj:    bigXj,
		ECDSAPub: ecdsaPub,
		Alias:    (*Alias)(data),
	})
}

func (data *publicFields) UnmarshalJSON(payload []byte) error {
	type Alias publicFields
	aux := &struct {
		BigXj    []*big.Int
		ECDSAPub []*big.Int
		*Alias
	}{
		Alias: (*Alias)(data),
	}
	if err := json.Unmarshal(payload, &aux); err != nil {
		return err
	}
	if bigXj, err := crypto.UnFlattenECPoints(tss.EC(), aux.BigXj); err == nil {
		data.BigXj = bigXj
	} else {
		return err
	}
	if pub, err := crypto.UnFlattenECPoints(tss.EC(), aux.ECDSAPub); err == nil && len(pub) == 1 {
		data.ECDSAPub = pub[0]
	} else {
		return err
	}
	return nil
}

// TssConfig + public fields
type secretConfig struct {
	SecretTssConfig *cryptoJSON `json:"config"` // encrypted tss config

	ListenAddr  string `json:"listen"`
	LogLevel    string `json:"log_level"`
	ProfileAddr string `json:"profile_addr"`
	Home        string
}

// Split LocalPartySaveData into priv.json and pub.json
// where priv.json is
func Save(keygenResult *keygen.LocalPartySaveData, nodeKey []byte, config KDFConfig, passphrase string, wPriv, wPub io.Writer) error {
	sFields := secretFields{
		keygenResult.Xi,
		keygenResult.PaillierSK,
		nodeKey,
	}

	priv, err := json.Marshal(sFields)
	if err != nil {
		return err
	}

	err = encryptAndWrite(priv, config, passphrase, wPriv)

	pFields := publicFields{
		keygenResult.ShareID,
		keygenResult.BigXj,
		keygenResult.ECDSAPub,
		keygenResult.PaillierPKs,
		keygenResult.NTildej,
		keygenResult.H1j,
		keygenResult.H2j,
		keygenResult.NTildei,
		keygenResult.H1i,
		keygenResult.H2i,
		keygenResult.Ks,
	}

	if pub, err := json.Marshal(&pFields); err == nil {
		return encryptAndWrite(pub, config, passphrase, wPub)
	} else {
		return err
	}
}

func SaveConfig(config *TssConfig, saveTo string) error {
	originalCfg, err := json.Marshal(config)
	if err != nil {
		return err
	}
	encrypted, err := encryptSecret(originalCfg, []byte(config.Password), config.KDFConfig)
	if err != nil {
		return err
	}
	sConfig := secretConfig{
		SecretTssConfig: encrypted,
		ListenAddr:      config.ListenAddr,
		LogLevel:        config.LogLevel,
		ProfileAddr:     config.ProfileAddr,
		Home:            config.Home,
	}

	bytes, err := json.MarshalIndent(sConfig, "", "    ")
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(path.Join(saveTo, "config.json"), bytes, os.FileMode(0600)); err != nil {
		return err
	}

	return nil
}

func Load(passphrase string, rPriv, rPub io.Reader) (saveData *keygen.LocalPartySaveData, nodeKey []byte, err error) {
	var sFields secretFields
	var pFields publicFields

	plainText, err := readAndDecrypt(rPriv, passphrase)
	if err != nil {
		return nil, nil, err
	}
	if err = json.Unmarshal(plainText, &sFields); err != nil {
		return nil, nil, err
	}

	plainText, err = readAndDecrypt(rPub, passphrase)
	if err != nil {
		return nil, nil, err
	}
	if err = json.Unmarshal(plainText, &pFields); err != nil {
		return nil, nil, err
	}

	return &keygen.LocalPartySaveData{
		LocalPreParams: keygen.LocalPreParams{
			PaillierSK: sFields.PaillierSk,
			NTildei:    pFields.NTildei,
			H1i:        pFields.H1i,
			H2i:        pFields.H2i,
		},
		LocalSecrets: keygen.LocalSecrets{
			Xi:      sFields.Xi,
			ShareID: pFields.ShareID,
		},

		BigXj:       pFields.BigXj,
		PaillierPKs: pFields.PaillierPks,

		NTildej:  pFields.NTildej,
		H1j:      pFields.H1j,
		H2j:      pFields.H2j,
		Ks:       pFields.Ks,
		ECDSAPub: pFields.ECDSAPub,
	}, sFields.NodeKey, nil
}

func LoadEcdsaPubkey(home, vault, passphrase string) (*ecdsa.PublicKey, error) {
	rPub, err := os.OpenFile(path.Join(home, vault, "pk.json"), os.O_RDONLY, 0400)
	if err != nil {
		return nil, err
	}
	defer rPub.Close()
	plaintext, err := readAndDecrypt(rPub, passphrase)
	if err != nil {
		return nil, err
	}
	var pFields publicFields
	if err := json.Unmarshal(plaintext, &pFields); err != nil {
		return nil, err
	}

	return &ecdsa.PublicKey{tss.EC(), pFields.ECDSAPub.X(), pFields.ECDSAPub.Y()}, nil
}

func LoadConfig(home, vault, passphrase string) (*TssConfig, error) {
	sConfigBytes, err := ioutil.ReadFile(path.Join(home, vault, "config.json"))
	if err != nil {
		return nil, err
	}
	var sConfig secretConfig
	if err := json.Unmarshal(sConfigBytes, &sConfig); err != nil {
		return nil, err
	}
	plaintext, err := decryptSecret(*sConfig.SecretTssConfig, passphrase)
	if err != nil {
		return nil, err
	}
	var config TssConfig
	if err := json.Unmarshal(plaintext, &config); err != nil {
		return nil, err
	} else {
		// let config in file override the encrypted fields, as user may manually change the exposed fields
		config.ListenAddr = sConfig.ListenAddr
		config.Home = sConfig.Home
		config.LogLevel = sConfig.LogLevel
		config.ProfileAddr = sConfig.ProfileAddr

		// assign kdf configs
		config.KDFConfig = sConfig.SecretTssConfig.KDFParams
		return &config, err
	}
}

func encryptAndWrite(src []byte, config KDFConfig, passphrase string, dest io.Writer) error {
	cryptoJson, err := encryptSecret(src, []byte(passphrase), config)
	if err != nil {
		return err
	}

	encrypted, err := json.Marshal(cryptoJson)
	if err != nil {
		return err
	}

	_, err = dest.Write(encrypted)
	if err != nil {
		return err
	}

	return nil
}

func readAndDecrypt(src io.Reader, passphrase string) ([]byte, error) {
	var encryptedSecret cryptoJSON
	sBytes, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to load bytes from file: %v", err)
	}

	err = json.Unmarshal(sBytes, &encryptedSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bytes: %v", err)
	}

	return decryptSecret(encryptedSecret, passphrase)
}

func encryptSecret(data, auth []byte, config KDFConfig) (*cryptoJSON, error) {
	salt := make([]byte, config.SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		Panic(fmt.Errorf("reading from crypto/rand failed: %v", err))
	}
	derivedKey := argon2.IDKey(auth, salt, config.Iterations, config.Memory, config.Parallelism, config.KeyLength)
	encryptKey := derivedKey[:len(derivedKey)-16]

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		Panic(fmt.Errorf("reading from crypto/rand failed: %v", err))
	}
	cipherText, err := aesCTRXOR(encryptKey, data, iv)
	if err != nil {
		return nil, err
	}

	d := sha3.New256()
	d.Write(derivedKey[len(derivedKey)-16:])
	d.Write(cipherText)
	mac := d.Sum(nil)

	config.Salt = hex.EncodeToString(salt)

	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	return &cryptoJSON{
		Cipher:       cipherAlg,
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    config,
		MAC:          hex.EncodeToString(mac),
	}, nil
}

func decryptSecret(encryptedSecret cryptoJSON, passphrase string) ([]byte, error) {
	if encryptedSecret.Cipher != cipherAlg {
		return nil, fmt.Errorf("Cipher not supported: %s", encryptedSecret.Cipher)
	}
	mac, err := hex.DecodeString(encryptedSecret.MAC)
	if err != nil {
		return nil, err
	}

	iv, err := hex.DecodeString(encryptedSecret.CipherParams.IV)
	if err != nil {
		return nil, err
	}

	cipherText, err := hex.DecodeString(encryptedSecret.CipherText)
	if err != nil {
		return nil, err
	}

	derivedKey, err := getKDFKey(encryptedSecret, passphrase)
	if err != nil {
		return nil, err
	}

	d := sha3.New256()
	d.Write(derivedKey[len(derivedKey)-16:])
	d.Write(cipherText)
	calculatedMAC := d.Sum(nil)

	if !bytes.Equal(calculatedMAC, mac) {
		return nil, errors.New("wrong vault passphrase")
	}

	plainText, err := aesCTRXOR(derivedKey[:len(derivedKey)-16], cipherText, iv)
	if err != nil {
		return nil, err
	}
	return plainText, err
}

func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	// AES-256 is selected due to size of encryptKey.
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

func getKDFKey(encryptedSecret cryptoJSON, auth string) ([]byte, error) {
	authArray := []byte(auth)
	salt, err := hex.DecodeString(encryptedSecret.KDFParams.Salt)
	if err != nil {
		return nil, err
	}
	dkLen := encryptedSecret.KDFParams.KeyLength
	i := encryptedSecret.KDFParams.Iterations
	m := encryptedSecret.KDFParams.Memory
	p := encryptedSecret.KDFParams.Parallelism
	return argon2.IDKey(authArray, salt, i, m, p, dkLen), nil
}
