package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
)

func init() {
	rootCmd.AddCommand(regroupCmd)
}

var regroupCmd = &cobra.Command{
	Use:   "regroup",
	Short: "regroup a new set of parties and threshold",
	Long:  "generate new_n secrete share with new_t threshold. At least old_t + 1 should participant",
	PreRun: func(cmd *cobra.Command, args []string) {
		vault := askVault()
		passphrase := askPassphrase()
		if err := common.ReadConfigFromHome(viper.GetViper(), false, viper.GetString(flagHome), vault, passphrase); err != nil {
			common.Panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var mustNew bool
		if _, err := os.Stat(path.Join(common.TssCfg.Home, common.TssCfg.Vault, "sk.json")); os.IsNotExist(err) {
			mustNew = true
		}

		if !mustNew {
			setIsOld()
			setIsNew()
		} else {
			common.TssCfg.IsOldCommittee = false
			common.TssCfg.IsNewCommittee = true
			setPassphrase()
			setOldN()
			setOldT()
		}
		setNewN()
		setNewT()

		var tssRegroup *exec.Cmd
		var tmpVault string
		if common.TssCfg.IsOldCommittee && common.TssCfg.IsNewCommittee {
			pwd, err := os.Getwd()
			if err != nil {
				common.Panic(err)
			}

			tmpVault = fmt.Sprintf("%s%s", common.TssCfg.Vault, common.RegroupSuffix)
			tmpMoniker := fmt.Sprintf("%s%s", common.TssCfg.Moniker, common.RegroupSuffix)
			devnull, err := os.Open(os.DevNull)
			if err != nil {
				common.Panic(err)
			}

			if _, err := os.Stat(path.Join(common.TssCfg.Home, tmpVault)); err == nil {
				os.RemoveAll(path.Join(common.TssCfg.Home, tmpVault))
			}

			// TODO: this relies on user doesn't rename the binary we released
			tssInit := exec.Command(path.Join(pwd, "tss"), "init", "--home", common.TssCfg.Home, "--vault_name", tmpVault, "--moniker", tmpMoniker, "--password", common.TssCfg.Password)
			tssInit.Stdin = devnull
			tssInit.Stdout = devnull

			if err := tssInit.Run(); err != nil {
				common.Panic(fmt.Errorf("failed to fork tss init command: %v", err))
			}

			setChannelId()
			setChannelPasswd()
			tssRegroup = exec.Command(path.Join(pwd, "tss"), "regroup", "--home", common.TssCfg.Home, "--vault_name", tmpVault, "--password", common.TssCfg.Password, "--parties", strconv.Itoa(common.TssCfg.Parties), "--threshold", strconv.Itoa(common.TssCfg.Threshold), "--new_parties", strconv.Itoa(common.TssCfg.NewParties), "--new_threshold", strconv.Itoa(common.TssCfg.NewThreshold), "--channel_password", common.TssCfg.ChannelPassword, "--channel_id", common.TssCfg.ChannelId, "--p2p.broadcast_sanity_check", strconv.FormatBool(common.TssCfg.BroadcastSanityCheck), "--log_level", common.TssCfg.LogLevel)
			stdOut, err := os.Create(path.Join(common.TssCfg.Home, tmpVault, "tss.log"))
			if err != nil {
				common.Panic(err)
			}
			tssRegroup.Stdin = devnull
			tssRegroup.Stdout = stdOut
			tssRegroup.Stderr = stdOut

			if err := tssRegroup.Start(); err != nil {
				common.Panic(fmt.Errorf("failed to fork tss regroup command: %v", err))
			}
		}

		common.TssCfg.BMode = common.PreRegroupMode
		bootstrapCmd.Run(cmd, args)
		common.TssCfg.BMode = common.RegroupMode

		c := client.NewTssClient(&common.TssCfg, client.RegroupMode, false)
		c.Start()

		if !common.TssCfg.IsOldCommittee {
			// delete tmp regroup suffix
			originExpectedNewPeers := make([]string, 0)
			for _, peer := range common.TssCfg.ExpectedNewPeers {
				moniker := p2p.GetMonikerFromExpectedPeers(peer)
				id := p2p.GetClientIdFromExpectedPeers(peer)
				moniker = strings.TrimSuffix(moniker, common.RegroupSuffix)
				originExpectedNewPeers = append(originExpectedNewPeers, fmt.Sprintf("%s@%s", moniker, id))
			}
			common.TssCfg.ExpectedPeers = originExpectedNewPeers
			common.TssCfg.PeerAddrs = common.TssCfg.NewPeerAddrs
			common.TssCfg.Parties = common.TssCfg.NewParties
			common.TssCfg.Threshold = common.TssCfg.NewThreshold
			common.TssCfg.NewParties = 0
			common.TssCfg.NewThreshold = 0
			common.TssCfg.Moniker = strings.TrimSuffix(common.TssCfg.Moniker, common.RegroupSuffix)
			originVault := common.TssCfg.Vault
			common.TssCfg.Vault = strings.TrimSuffix(common.TssCfg.Vault, common.RegroupSuffix)
			updateConfigForRegroup(originVault)
		}

		if !mustNew && common.TssCfg.IsNewCommittee && tssRegroup != nil {
			err := tssRegroup.Wait()
			if err != nil {
				client.Logger.Error(fmt.Errorf("failed to wait child tss process finished: %v", err))
			}

			// TODO: Make sure this works under different os (linux and windows)
			backupPath := path.Join(common.TssCfg.Home, common.TssCfg.Vault+"_tgbak")

			err = os.Rename(
				path.Join(common.TssCfg.Home, common.TssCfg.Vault),
				backupPath,
			)
			if err != nil {
				client.Logger.Error(err)
			}

			err = os.Rename(
				path.Join(common.TssCfg.Home, tmpVault),
				path.Join(common.TssCfg.Home, common.TssCfg.Vault))
			if err != nil {
				client.Logger.Error(err)
			}

			err = os.RemoveAll(backupPath)
			if err != nil {
				client.Logger.Error(err)
			}
			client.Logger.Info("secret share and configuration has been updated")
		}

		if mustNew {
			addToBnbcli(c.PubKey())
		}
	},
}

func setIsOld() {
	if common.TssCfg.IsOldCommittee {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	answer, err := common.GetBool("Participant as a old committee?[Y/n]:", true, reader)
	if err != nil {
		common.Panic(err)
	}
	if answer {
		common.TssCfg.IsOldCommittee = true
	}
}

func setIsNew() {
	if common.TssCfg.IsNewCommittee {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	answer, err := common.GetBool("Participant as a new committee?[Y/n]:", true, reader)
	if err != nil {
		common.Panic(err)
	}
	if answer {
		common.TssCfg.IsNewCommittee = true
	}
}

func setOldN() {
	if common.TssCfg.Parties > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	n, err := common.GetInt("please set old total parties(n) (default: 3): ", 3, reader)
	if err != nil {
		common.Panic(err)
	}
	if n <= 1 {
		common.Panic(fmt.Errorf("n should greater than 1"))
	}
	common.TssCfg.Parties = n
}

func setOldT() {
	if common.TssCfg.Threshold > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	t, err := common.GetInt("please set old threshold(t), at least t + 1 parties needs participant signing (default: 1): ", 1, reader)
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

func setNewN() {
	if common.TssCfg.NewParties > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	n, err := common.GetInt("please set new total parties(n) (default 3): ", 3, reader)
	if err != nil {
		common.Panic(err)
	}
	if n <= 1 {
		common.Panic(fmt.Errorf("n should greater than 1"))
	}
	common.TssCfg.NewParties = n
}

func setNewT() {
	if common.TssCfg.NewThreshold > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	t, err := common.GetInt("please set new threshold(t), at least t + 1 parties needs participant signing (default: 1): ", 1, reader)
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
	common.TssCfg.NewThreshold = t
}
