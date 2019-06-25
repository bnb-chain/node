package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/tendermint/iavl"
	"github.com/tendermint/tendermint/cmd/tendermint/commands"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/lite/proxy"
	rpcclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/cosmos/cosmos-sdk/store"
)

var logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))

func main() {
	if err := LiteCmd.Execute(); err != nil {
		panic(err)
	}
}

// LiteCmd represents the base command when called without any subcommands
var LiteCmd = &cobra.Command{
	Use:   "lite",
	Short: "Run lite-client proxy server, verifying binance rpc",
	Long: `This node will run a secure proxy to a binance rpc server.

All calls that can be tracked back to a block header by a proof
will be verified before passing them back to the caller. Other that
that it will present the same interface as a full binance node,
just with added trust and running locally.`,
	RunE:         runProxy,
	SilenceUsage: true,
}

var (
	listenAddr         string
	nodeAddr           string
	chainID            string
	home               string
	maxOpenConnections int
	cacheSize          int
)

func init() {
	LiteCmd.Flags().StringVar(&listenAddr, "laddr", "tcp://localhost:27147", "Serve the proxy on the given address")
	LiteCmd.Flags().StringVar(&nodeAddr, "node", "tcp://localhost:27147", "Connect to a binance node at this address")
	LiteCmd.Flags().StringVar(&chainID, "chain-id", "bnbchain", "Specify the binance chain ID")
	LiteCmd.Flags().StringVar(&home, "home-dir", ".binance-lite", "Specify the home directory")
	LiteCmd.Flags().IntVar(&maxOpenConnections, "max-open-connections", 900, "Maximum number of simultaneous connections (including WebSocket).")
	LiteCmd.Flags().IntVar(&cacheSize, "cache-size", 10, "Specify the memory trust store cache size")
}

func runProxy(cmd *cobra.Command, args []string) error {
	cmn.TrapSignal(logger, func() {
		// TODO: close up shop
	})

	nodeAddr, err := commands.EnsureAddrHasSchemeOrDefaultToTCP(nodeAddr)
	if err != nil {
		return err
	}
	listenAddr, err := commands.EnsureAddrHasSchemeOrDefaultToTCP(listenAddr)
	if err != nil {
		return err
	}

	// First, connect a client
	logger.Info("Connecting to source HTTP client...")
	node := rpcclient.NewHTTP(nodeAddr, "/websocket")

	logger.Info("Constructing Verifier...")
	cert, err := proxy.NewVerifier(chainID, home, node, logger, cacheSize)
	if err != nil {
		return cmn.ErrorWrap(err, "constructing Verifier")
	}
	cert.SetLogger(logger)
	sc := proxy.SecureClient(node, cert)
	sc.RegisterOpDecoder(store.ProofOpMultiStore, store.MultiStoreProofOpDecoder)
	sc.RegisterOpDecoder(
		iavl.ProofOpIAVLValue,
		iavl.IAVLValueOpDecoder,
	)

	logger.Info("Starting proxy...")
	err = proxy.StartProxy(sc, listenAddr, logger, maxOpenConnections)
	if err != nil {
		return cmn.ErrorWrap(err, "starting proxy")
	}

	select {}
}
