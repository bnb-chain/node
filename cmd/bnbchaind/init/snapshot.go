package init

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tendermint/tendermint/blockchain"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/snapshot"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common"
)

const (
	flagHeight = "height"
)

func SnapshotCmd(ctx *server.Context, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Take a snapshot for state sync",
		RunE: func(_ *cobra.Command, _ []string) error {
			logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
			config := ctx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))

			logger.Info("setup block db")
			blockDB, err := node.DefaultDBProvider(&node.DBContext{"blockstore", config})
			if err != nil {
				return err
			}

			logger.Info("setup state db")
			stateDB, err := node.DefaultDBProvider(&node.DBContext{"state", config})
			if err != nil {
				return err
			}

			logger.Info("setup tx db")
			txDB, err := node.DefaultDBProvider(&node.DBContext{"tx_index", config})
			if err != nil {
				return err
			}

			logger.Info("setup application db")
			appDB, err := node.DefaultDBProvider(&node.DBContext{"application", config})
			if err != nil {
				return err
			}

			logger.Info("build cms")
			cms := store.NewCommitMultiStore(appDB)
			for _, name := range common.StoreKeyNames {
				cms.MountStoreWithDB(common.StoreKeyNameMap[name], sdk.StoreTypeIAVL, nil)
			}
			cms.MountStoreWithDB(common.TParamsStoreKey, sdk.StoreTypeTransient, nil)
			cms.MountStoreWithDB(common.TStakeStoreKey, sdk.StoreTypeTransient, nil)

			logger.Info("load latest version")
			if err := cms.LoadLatestVersion(); err != nil {
				return err
			}

			snapshot.InitSnapshotManager(
				stateDB,
				txDB,
				blockchain.NewBlockStore(blockDB),
				config.DBDir(),
				logger)

			helper := store.NewStateSyncHelper(logger, appDB, cms, cdc)

			logger.Info("start take snapshot")
			helper.ReloadSnapshotRoutine(viper.GetInt64(flagHeight), 0)

			return nil
		},
	}

	cmd.Flags().Int64(flagHeight, 0, "specify a syncable height (the height must haven't been pruned")
	cmd.MarkFlagRequired(flagHeight)

	return cmd
}
