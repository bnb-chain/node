package cmd

import (
	"os"
	"path"

	"github.com/ipfs/go-log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
)

const (
	flagHome   = "home"
	flagVault  = "vault_name"
	flagPrefix = "address_prefix"
)

var rootCmd = &cobra.Command{
	Use:   "tss",
	Short: "Threshold signing scheme",
	Long:  `Complete documentation is available at https://github.com/binance-chain/tss`, // TODO: replace documentation here
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlags(cmd.Flags())
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
	},
}

func Execute() {
	initConfigAndLogLevel()

	if err := rootCmd.Execute(); err != nil {
		client.Logger.Error(err)
		os.Exit(1)
	}
}

func initConfigAndLogLevel() {
	bindP2pConfigs()
	bindKdfConfigs()
	bindClientConfigs()

	home, err := os.UserHomeDir()
	if err != nil {
		common.Panic(err)
	}
	rootCmd.PersistentFlags().String(flagHome, path.Join(home, ".tss"), "Path to config/route_table/node_key/tss_key files, configs in config file can be overridden by command line arg quments")
}

func bindP2pConfigs() {
	initCmd.PersistentFlags().String("p2p.listen", "", "Adds a multiaddress to the listen list")
	//rootCmd.PersistentFlags().StringSlice("p2p.bootstraps", []string{}, "bootstrap server list in multiaddr format, i.e. /ip4/127.0.0.1/tcp/27148/p2p/12D3KooWMXTGW6uHbVs7QiHEYtzVa4RunbugxRcJhGU43qAvfAa1")
	//rootCmd.PersistentFlags().StringSlice("p2p.relays", []string{}, "relay server list")
	keygenCmd.PersistentFlags().StringSlice("p2p.peer_addrs", []string{}, "peer's multiple addresses")
	regroupCmd.PersistentFlags().StringSlice("p2p.new_peer_addrs", []string{}, "unknown peer's multiple addresses")
	//rootCmd.PersistentFlags().StringSlice("p2p.peers", []string{}, "peers in this threshold scheme")
	//rootCmd.PersistentFlags().Bool("p2p.default_bootstrap", false, "whether to use default bootstrap")
}

// more detail explanation of these parameters can be found:
// https://github.com/P-H-C/phc-winner-argon2/blob/master/argon2-specs.pdf
// https://www.alexedwards.net/blog/how-to-hash-and-verify-passwords-with-argon2-in-go
func bindKdfConfigs() {
	initCmd.PersistentFlags().Uint32("kdf.memory", 65536, "The amount of memory used by the algorithm (in kibibytes)")
	initCmd.PersistentFlags().Uint32("kdf.iterations", 13, "The number of iterations (or passes) over the memory.")
	initCmd.PersistentFlags().Uint8("kdf.parallelism", 4, "The number of threads (or lanes) used by the algorithm.")
	initCmd.PersistentFlags().Uint32("kdf.salt_length", 16, "Length of the random salt. 16 bytes is recommended for password hashing.")
	initCmd.PersistentFlags().Uint32("kdf.key_length", 48, "Length of the generated key (or password hash). must be 32 bytes or more")
}

func bindClientConfigs() {
	initCmd.PersistentFlags().String("moniker", "", "moniker of current party")
	rootCmd.PersistentFlags().String(flagVault, "", "name of vault of this party")
	keygenCmd.PersistentFlags().String(flagPrefix, "bnb", "prefix of bech32 address")
	describeCmd.PersistentFlags().String(flagPrefix, "bnb", "prefix of bech32 address")
	keygenCmd.PersistentFlags().Int("threshold", 0, "threshold of this scheme")
	regroupCmd.PersistentFlags().Int("threshold", 0, "threshold of this scheme")
	keygenCmd.PersistentFlags().Int("parties", 0, "total parities of this scheme")
	regroupCmd.PersistentFlags().Int("parties", 0, "total parities of this scheme")
	regroupCmd.PersistentFlags().Int("new_threshold", 0, "new threshold of regrouped scheme")
	regroupCmd.PersistentFlags().Int("new_parties", 0, "new total parties of regrouped scheme")
	rootCmd.PersistentFlags().String("password", "", "password, should only be used for testing. If empty, you will be prompted for password to save/load the secret/public share and config")
	signCmd.PersistentFlags().String("message", "", "message(in *big.Int.String() format) to be signed, only used in sign mode")
	rootCmd.PersistentFlags().String("log_level", "info", "log level")

	keygenCmd.PersistentFlags().Bool("p2p.broadcast_sanity_check", true, "whether verify broadcast message's hash with peers")
	signCmd.PersistentFlags().Bool("p2p.broadcast_sanity_check", true, "whether verify broadcast message's hash with peers")
	regroupCmd.PersistentFlags().Bool("p2p.broadcast_sanity_check", true, "whether verify broadcast message's hash with peers")

	keygenCmd.PersistentFlags().String("channel_id", "", "channel id of this session")
	signCmd.PersistentFlags().String("channel_id", "", "channel id of this session")
	regroupCmd.PersistentFlags().String("channel_id", "", "channel id of this session")

	keygenCmd.PersistentFlags().String("channel_password", "", "channel password of this session")
	signCmd.PersistentFlags().String("channel_password", "", "channel password of this session")
	regroupCmd.PersistentFlags().String("channel_password", "", "channel password of this session")

	channelCmd.PersistentFlags().Int("channel_expire", 0, "expire time in minutes of this channel")

	regroupCmd.PersistentFlags().Bool("is_old", false, "whether this party is an old committee. If it is set to true, it will participant signing in regroup. There should be only t+1 parties set this to true for one regroup")
	regroupCmd.PersistentFlags().Bool("is_new_member", false, "whether this party is new committee, for new party it will changed to true automatically. if an old party set this to true, its share will be replaced by one generated one")
}

func initLogLevel(cfg common.TssConfig) {
	log.SetLogLevel("tss", cfg.LogLevel)
	log.SetLogLevel("tss-lib", cfg.LogLevel)
	log.SetLogLevel("srv", cfg.LogLevel)
	log.SetLogLevel("trans", cfg.LogLevel)
	log.SetLogLevel("p2p_utils", cfg.LogLevel)
	log.SetLogLevel("common", cfg.LogLevel)

	// libp2p loggers
	log.SetLogLevel("dht", "error")
	log.SetLogLevel("discovery", "error")
	log.SetLogLevel("swarm2", "error")
	log.SetLogLevel("stream-upgrader", "error")
}
