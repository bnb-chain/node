package cmd

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/common"
)

func init() {
	rootCmd.AddCommand(channelCmd)
}

var channelCmd = &cobra.Command{
	Use:              "channel",
	Short:            "generate a channel id for bootstrapping",
	TraverseChildren: false, // TODO: figure out how to disable parent's options
	Run: func(cmd *cobra.Command, args []string) {
		channelId, err := rand.Int(rand.Reader, big.NewInt(999))
		if err != nil {
			common.Panic(err)
		}
		expire := askChannelExpire()
		expireTime := time.Now().Add(time.Duration(expire) * time.Minute).Unix()
		fmt.Printf("channel id: %s\n", fmt.Sprintf("%.3d%s", channelId.Int64(), common.ConvertTimestampToHex(expireTime)))
	},
}

func askChannelExpire() int {
	if viper.GetInt("channel_expire") > 0 {
		return viper.GetInt("channel_expire")
	}

	reader := bufio.NewReader(os.Stdin)
	expire, err := common.GetInt("please set expire time in minutes, (default: 30): ", 30, reader)
	if err != nil {
		common.Panic(err)
	}
	if expire <= 0 {
		common.Panic(fmt.Errorf("expire time should not be zero or negative value"))
	}
	return expire
}
