package init

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	abciServer "github.com/tendermint/tendermint/abci/server"
	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/mock"

	"github.com/bnb-chain/node/app"
)

func TestInitCmd(t *testing.T) {
	defer server.SetupViper(t)()
	defer setupClientHome(t)()

	logger := log.NewNopLogger()
	cfg, err := tcmd.ParseConfig()
	require.Nil(t, err)
	ctx := server.NewContext(cfg, logger)
	cdc := app.MakeCodec()
	appInit := server.AppInit{
		AppGenTx:    mock.AppGenTx,
		AppGenState: mock.AppGenState,
	}
	cmd := InitCmd(ctx, cdc, appInit)
	err = cmd.RunE(nil, nil)
	require.NoError(t, err)
}

func setupClientHome(t *testing.T) func() {
	clientDir, err := os.MkdirTemp("", "mock-sdk-cmd")
	require.Nil(t, err)
	viper.Set(flagClientHome, clientDir)
	viper.Set(flagOverwriteKey, true)
	return func() {
		if err := os.RemoveAll(clientDir); err != nil {
			// TODO: Handle with #870
			panic(err)
		}
	}
}

func TestEmptyState(t *testing.T) {
	defer server.SetupViper(t)()
	defer setupClientHome(t)()
	logger := log.NewNopLogger()
	cfg, err := tcmd.ParseConfig()
	require.Nil(t, err)
	ctx := server.NewContext(cfg, logger)
	cdc := app.MakeCodec()
	appInit := server.AppInit{
		AppGenTx:    mock.AppGenTx,
		AppGenState: mock.AppGenStateEmpty,
	}
	cmd := InitCmd(ctx, cdc, appInit)
	err = cmd.RunE(nil, nil)
	require.NoError(t, err)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cmd = server.ExportCmd(ctx, cdc, nil)
	err = cmd.RunE(nil, nil)
	require.NoError(t, err)

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	w.Close()
	os.Stdout = old
	out := <-outC
	require.Contains(t, out, "WARNING: State is not initialized")
	require.Contains(t, out, "genesis_time")
	require.Contains(t, out, "chain_id")
	require.Contains(t, out, "consensus_params")
	require.Contains(t, out, "app_hash")
}

func TestStartStandAlone(t *testing.T) {
	home, err := os.MkdirTemp("", "mock-sdk-cmd")
	require.Nil(t, err)
	defer func() {
		os.RemoveAll(home)
	}()
	viper.Set(cli.HomeFlag, home)
	viper.Set(client.FlagName, "moniker")
	defer setupClientHome(t)()

	logger := log.NewNopLogger()
	cfg, err := tcmd.ParseConfig()
	require.Nil(t, err)
	ctx := server.NewContext(cfg, logger)
	cdc := app.MakeCodec()
	appInit := server.AppInit{
		AppGenTx:    mock.AppGenTx,
		AppGenState: mock.AppGenState,
	}
	initCmd := InitCmd(ctx, cdc, appInit)
	err = initCmd.RunE(nil, nil)
	require.NoError(t, err)

	app, err := mock.NewApp(home, logger)
	require.Nil(t, err)
	svrAddr, _, err := server.FreeTCPAddr()
	require.Nil(t, err)
	svr, err := abciServer.NewServer(svrAddr, "socket", app)
	require.Nil(t, err, "error creating listener")
	svr.SetLogger(logger.With("module", "abci-server"))
	svr.Start()

	timer := time.NewTimer(time.Duration(2) * time.Second)
	select {
	case <-timer.C:
		svr.Stop()
	}
}

func TestInitNodeValidatorFiles(t *testing.T) {
	home, err := os.MkdirTemp("", "mock-sdk-cmd")
	require.Nil(t, err)
	defer func() {
		os.RemoveAll(home)
	}()
	viper.Set(cli.HomeFlag, home)
	viper.Set(client.FlagName, "moniker")
	cfg, err := tcmd.ParseConfig()
	require.Nil(t, err)
	nodeID, valPubKey := InitializeNodeValidatorFiles(cfg)
	require.NotEqual(t, "", nodeID)
	require.NotEqual(t, 0, len(valPubKey.Bytes()))
}
