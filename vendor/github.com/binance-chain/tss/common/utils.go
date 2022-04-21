package common

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/ipfs/go-log"
	"github.com/mattn/go-isatty"
)

var logger = log.Logger("common")

const (
	RegroupSuffix = "_rgtmp"
)

func Encrypt(passphrase string, param PeerParam) ([]byte, error) {
	text, err := json.Marshal(param)
	key := sha256.Sum256([]byte(passphrase))

	// generate a new aes cipher using our 32 byte long key
	c, err := aes.NewCipher(key[:])
	// if there are any errors, handle them
	if err != nil {
		return nil, err
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	// if any error generating new GCM
	// handle them
	if err != nil {
		return nil, err
	}

	// creates a new byte array the size of the nonce
	// which must be passed to Seal
	nonce := make([]byte, gcm.NonceSize())
	// populates our nonce with a cryptographically secure
	// random sequence
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	return gcm.Seal(nonce, nonce, text, nil), nil
}

func Decrypt(ciphertext []byte, channelId, passphrase string) (param *PeerParam, error error) {
	key := sha256.Sum256([]byte(passphrase))
	c, err := aes.NewCipher(key[:])
	if err != nil {
		error = err
		return
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		error = err
		return
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		error = fmt.Errorf("ciphertext is not as long as expected")
		return
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		error = err
		return
	}

	param = &PeerParam{}
	error = json.Unmarshal(plaintext, param)
	if error != nil {
		return
	}
	if param.ChannelId != channelId {
		error = fmt.Errorf("wrong channel id of message")
		return
	}
	epochSeconds := ConvertHexToTimestamp(channelId[3:])
	if time.Now().Unix() > int64(epochSeconds) {
		Panic(fmt.Errorf("channel id has been expired, please regenerate a new one"))
	}
	return param, nil
}

// conversion between hex and int32 epoch seconds
// refer: https://www.epochconverter.com/hex
func ConvertTimestampToHex(timestamp int64) string {
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.BigEndian, int32(timestamp)); err != nil {
		return ""
	}
	return fmt.Sprintf("%X", buf.Bytes())
}

// conversion between hex and int32 epoch seconds
// refer: https://www.epochconverter.com/hex
func ConvertHexToTimestamp(hexTimestamp string) int {
	dst := make([]byte, 8)
	hex.Decode(dst, []byte(hexTimestamp))
	var epochSeconds int32
	if err := binary.Read(bytes.NewReader(dst), binary.BigEndian, &epochSeconds); err != nil {
		return math.MaxInt64
	}
	return int(epochSeconds)
}

func ReplaceIpInAddr(addr, realIp string) string {
	re := regexp.MustCompile(`((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\/tcp`)
	return re.ReplaceAllString(addr, realIp+"/tcp")
}

func ConvertMultiAddrStrToNormalAddr(listenAddr string) (string, error) {
	re := regexp.MustCompile(`((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\/tcp\/([0-9]+)`)
	all := re.FindStringSubmatch(listenAddr)
	if len(all) != 6 {
		return "", fmt.Errorf("failed to convert multiaddr to listen addr")
	}
	return fmt.Sprintf("%s:%s", all[1], all[5]), nil
}

func GetInt(prompt string, defaultValue int, buf *bufio.Reader) (int, error) {
	s, err := GetString(prompt, buf)
	if err != nil {
		return 0, err
	}
	if s == "" {
		return defaultValue, nil
	}
	res, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%s is an invalid integer", s)
	} else {
		return res, nil
	}
}

// GetString simply returns the trimmed string output of a given reader.
func GetString(prompt string, buf *bufio.Reader) (string, error) {
	if inputIsTty() && prompt != "" {
		PrintPrefixed(prompt)
	}

	out, err := readLineFromBuf(buf)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func GetBool(prompt string, defaultValue bool, buf *bufio.Reader) (bool, error) {
	answer, err := GetString(prompt, buf)
	if err != nil {
		return false, err
	}
	if answer == "y" || answer == "Y" || answer == "Yes" || answer == "YES" {
		return true, nil
	} else if answer == "n" || answer == "N" || answer == "No" || answer == "NO" {
		return false, nil
	} else if strings.TrimSpace(answer) == "" {
		return defaultValue, nil
	} else {
		return false, fmt.Errorf("input does not make sense, please input 'y' or 'n'")
	}
}

func Panic(err error) {
	logger.Error(err)
	trace := fmt.Sprintf("stack:\n%v", string(debug.Stack()))
	logger.Error(trace)
	os.Exit(1)
}

// during bootstrapping ssdp service will connect to test availability and disconnect directly
// we should skip tcp close write/read error for such connections
func SkipTcpClosePanic(err error) {
	if !strings.Contains(err.Error(), "connection reset by peer") && !strings.Contains(err.Error(), "EOF") {
		Panic(err)
	}
}

// inputIsTty returns true iff we have an interactive prompt,
// where we can disable echo and request to repeat the password.
// If false, we can optimize for piped input from another command
func inputIsTty() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

// readLineFromBuf reads one line from stdin.
// Subsequent calls reuse the same buffer, so we don't lose
// any input when reading a password twice (to verify)
func readLineFromBuf(buf *bufio.Reader) (string, error) {
	pass, err := buf.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(pass), nil
}

// PrintPrefixed prints a string with > prefixed for use in prompts.
func PrintPrefixed(msg string) {
	msg = fmt.Sprintf("> %s\n", msg)
	fmt.Fprint(os.Stderr, msg)
}
