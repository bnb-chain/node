package cmd

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/bgentry/speakeasy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
)

func init() {
	rootCmd.AddCommand(keygenCmd)
}

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "key generation",
	Long:  "generate secret share of t of n scheme",
	PreRun: func(cmd *cobra.Command, args []string) {
		vault := askVault()
		passphrase := askPassphrase()
		if err := common.ReadConfigFromHome(viper.GetViper(), false, viper.GetString(flagHome), vault, passphrase); err != nil {
			common.Panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		checkOverride()
		setN()
		setT()
		bootstrapCmd.Run(cmd, args)
		checkN()
		setPassphrase()
		c := client.NewTssClient(&common.TssCfg, client.KeygenMode, false)
		c.Start()

		updateConfig()
		addToBnbcli(c.PubKey())
	},
}

func checkOverride() {
	if _, err := os.Stat(path.Join(common.TssCfg.Home, common.TssCfg.Vault, "sk.json")); err == nil {
		// we have already done keygen before
		reader := bufio.NewReader(os.Stdin)
		answer, err := common.GetBool("Vault already generated, do you like override it[y/N]: ", false, reader)
		if err != nil {
			common.Panic(err)
		}
		if !answer {
			client.Logger.Info("nothing happened")
			os.Exit(0)
		} else {
			common.TssCfg.Parties = viper.GetInt("parties")
			common.TssCfg.Threshold = viper.GetInt("threshold")
		}
	}
}

func checkN() {
	if common.TssCfg.Parties > 0 && len(common.TssCfg.ExpectedPeers) != common.TssCfg.Parties-1 {
		common.Panic(fmt.Errorf("peers are not correctly set during bootstrap"))
	}
}

func setN() {
	if common.TssCfg.Parties > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	n, err := common.GetInt("please set total parties(n) (default: 3): ", 3, reader)
	if err != nil {
		common.Panic(err)
	}
	if n <= 1 {
		common.Panic(fmt.Errorf("n should greater than 1"))
	}
	common.TssCfg.Parties = n
}

func setT() {
	if common.TssCfg.Threshold > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	t, err := common.GetInt("please set threshold(t), at least t + 1 parties needs participant signing (default: 1): ", 1, reader)
	if err != nil {
		common.Panic(err)
	}
	if t <= 0 {
		common.Panic(fmt.Errorf("t should greater than 0"))
	}
	// we allowed t+1 == n, for most common use case 2-2 scheme
	if t+1 > common.TssCfg.Parties {
		common.Panic(fmt.Errorf("t + 1 should less than or equals to parties"))
	}
	common.TssCfg.Threshold = t
}

func askPassphrase() string {
	if pw := viper.GetString("password"); pw != "" {
		checkComplexityOfPassword(pw)
		return pw
	}

	if p, err := speakeasy.Ask("> Password to sign with this vault:"); err == nil {
		viper.Set("password", p)
		checkComplexityOfPassword(p)
		return p
	} else {
		common.Panic(err)
		return ""
	}
}

// CapturingPassThroughWriter is a writer that remembers
// data written to it and passes it to w
type CapturingPassThroughWriter struct {
	buf bytes.Buffer
	w   io.Writer
}

// NewCapturingPassThroughWriter creates new CapturingPassThroughWriter
func NewCapturingPassThroughWriter(w io.Writer) *CapturingPassThroughWriter {
	return &CapturingPassThroughWriter{
		w: w,
	}
}
func (w *CapturingPassThroughWriter) Write(d []byte) (int, error) {
	w.buf.Write(d)
	if strings.Contains(string(w.Bytes()), "ERROR: resource temporarily unavailable") {
		return len(d), nil
	}
	return w.w.Write(d)
}

// Bytes returns bytes written to the writer
func (w *CapturingPassThroughWriter) Bytes() []byte {
	return w.buf.Bytes()
}

// TODO: bad smell of this method!!! it means tss relies on cosmos (cyclic dependency)
func addToBnbcli(pubKey crypto.PubKey) {
	client.Logger.Infof("trying to add the key to bnbcli's default keystore...")
	// invoke bnbcli add the generated key into bnbcli's keystore
	bnbcliName := fmt.Sprintf("tss_%s", common.TssCfg.Moniker)
	if common.TssCfg.Vault != "" {
		bnbcliName += "_" + common.TssCfg.Vault
	}
	pwd, err := os.Getwd()
	if err != nil {
		common.Panic(err)
	}
	execuable := "tbnbcli"
	if common.TssCfg.AddressPrefix == "bnb" {
		if _, err := os.Stat(path.Join(pwd, "bnbcli")); err == nil {
			execuable = "bnbcli"
		}
	}

	// TODO: support other types key
	pubKeyBytes := pubKey.(secp256k1.PubKeySecp256k1)
	pubKeyHex := hex.EncodeToString(pubKeyBytes[:])

	interactive := bytes.NewBuffer(make([]byte, 0))
	go func() {
		for {
			interactive.WriteString("y\n")
			time.Sleep(1 * time.Second)
		}
	}()

	retry := 3
	for tried := 0; tried < retry; tried++ {
		bnbcli := exec.Command(path.Join(pwd, execuable), "keys", "add", "--tss", "-t", "tss", "--tss-home", common.TssCfg.Home, "--tss-vault", common.TssCfg.Vault, "--tss-pubkey", pubKeyHex, bnbcliName)
		stdoutIn, _ := bnbcli.StdoutPipe()
		stderrIn, _ := bnbcli.StderrPipe()
		bnbcli.Stdin = interactive
		stdout := NewCapturingPassThroughWriter(os.Stdout)
		stderr := NewCapturingPassThroughWriter(os.Stderr)

		var errStdout, errStderr error
		go func() {
			_, errStdout = io.Copy(stdout, stdoutIn)
		}()
		go func() {
			_, errStderr = io.Copy(stderr, stderrIn)
		}()

		err = bnbcli.Start()
		if err != nil {
			common.Panic(err)
		}
		err = bnbcli.Wait()

		if strings.Contains(string(stderr.Bytes()), "ERROR: resource temporarily unavailable") {
			time.Sleep(time.Second)
			tried-- // contention doesn't count real failure
		} else {
			if err != nil {
				client.Logger.Errorf("%s failed with %v\n", execuable, err)
				if tried == retry-1 {
					cmd := fmt.Sprintf("%s keys add --tss -t tss --tss-home %s --tss-vault %s %s", execuable, common.TssCfg.Home, common.TssCfg.Vault, bnbcliName)
					client.Logger.Infof("Cannot add tss key to %s's default keystore, please try this command manually: %s", execuable, cmd)
				}
			} else {
				client.Logger.Infof("added %s to bnbcli's default keystore", bnbcliName)
				break
			}
		}
	}
}

func checkComplexityOfPassword(p string) {
	if len(p) <= 8 {
		common.Panic(fmt.Errorf("password is too simple, should be longer than 8 characters"))
	}
}
