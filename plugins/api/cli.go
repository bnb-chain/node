package api

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	tmserver "github.com/tendermint/tendermint/rpc/lib/server"

	sdk "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/wire"
)

// ServeCommand will generate a long-running rest server
// that exposes functionality similar to the cli, but over http
func ServeCommand(cdc *wire.Codec) *cobra.Command {
	flagListenAddr := "laddr"
	flagMaxOpenConnections := "max-open"

	cmd := &cobra.Command{
		Use:   "api-server",
		Short: "Start the API server daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(types.GetAccountDecoder(cdc))
			listenAddr := viper.GetString(flagListenAddr)
			server := newServer(ctx, cdc).bindRoutes()
			handler := server.router
			logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "apiserv")
			maxOpen := viper.GetInt(flagMaxOpenConnections)

			cfg := &tmserver.Config{MaxOpenConnections: maxOpen}
			listener, err := tmserver.Listen(listenAddr, cfg)
			if err != nil {
				return err
			}
			go func() {
				// wrap to handle the error
				err := tmserver.StartHTTPServer(listener, handler, logger, cfg)
				if err != nil {
					panic(err)
				}
			}()

			logger.Info("REST server started")

			// wait forever and cleanup
			cmn.TrapSignal(logger, func() {
				err := listener.Close()
				if err != nil {
					logger.Error("error closing listener", "err", err)
				}
			})
			select {}
		},
	}

	cmd.Flags().String(flagListenAddr, "tcp://localhost:8080", "The address for the server to listen on")
	cmd.Flags().String(sdk.FlagChainID, "", "The chain ID to connect to")
	cmd.Flags().String(sdk.FlagNode, "tcp://localhost:26657", "Address of the node to connect to")
	cmd.Flags().Int(flagMaxOpenConnections, 1000, "The number of maximum open connections")
	cmd.Flags().Bool(sdk.FlagTrustNode, true, "Trust connected full node (don't verify proofs for responses)")

	return cmd
}
