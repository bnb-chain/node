package app

import (
	"encoding/json"
	"errors"
	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
	"github.com/cosmos/cosmos-sdk/server"
	serverCfg "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/tendermint/tendermint/crypto"
	tmtypes "github.com/tendermint/tendermint/types"
	"os"
	"path/filepath"
)

type GenesisState struct {
	Tokens     []types.Token    `json:"tokens"`
	Accounts   []GenesisAccount `json:"accounts"`
	DexGenesis dex.Genesis      `json:"dex"`
}

// GenesisAccount doesn't need pubkey or sequence
type GenesisAccount struct {
	Name    string         `json:"name"`
	Address sdk.AccAddress `json:"address"`
	ValAddr crypto.Address `json:"valaddr"`
}

// NewGenesisAccount -
func NewGenesisAccount(aa *types.AppAccount) GenesisAccount {
	return GenesisAccount{
		Name:    aa.Name,
		Address: aa.GetAddress(),
	}
}

// convert GenesisAccount to AppAccount
func (ga *GenesisAccount) ToAppAccount() (acc *types.AppAccount) {
	baseAcc := auth.BaseAccount{
		Address: ga.Address,
	}
	return &types.AppAccount{
		BaseAccount: baseAcc,
		Name:        ga.Name,
	}
}

func BinanceAppInit() server.AppInit {
	return server.AppInit{
		AppGenTx:    BinanceAppGenTx,
		AppGenState: BinanceAppGenState,
	}
}

type GenTx struct {
	Name    string         `json:"name"`
	Address sdk.AccAddress `json:"address"`
	PubKey  crypto.PubKey  `json:"pub_key"`
}

func BinanceAppGenTx(cdc *wire.Codec, pk crypto.PubKey, genTxConfig serverCfg.GenTx) (
	appGenTx, cliPrint json.RawMessage, validator tmtypes.GenesisValidator, err error) {

	// write app.toml when we run testnet command, we only know the `current` rootDir for each validator here
	// otherwise, we can only generate at ~/.bnbchaind/config/app.toml
	appConfigFilePath := filepath.Join(ServerContext.Context.Config.RootDir, "config/", config.AppConfigFileName+".toml")
	if _, err := os.Stat(appConfigFilePath); os.IsNotExist(err) {
		config.WriteConfigFile(appConfigFilePath, ServerContext.BinanceChainConfig)
	}

	if genTxConfig.Name == "" {
		return nil, nil, tmtypes.GenesisValidator{}, errors.New("Must specify --name (validator moniker)")
	}

	var addr sdk.AccAddress
	var secret string
	addr, secret, err = server.GenerateSaveCoinKey(genTxConfig.CliRoot, genTxConfig.Name, "1234567890", genTxConfig.Overwrite)
	if err != nil {
		return
	}

	cliPrint, err = makePrintMessage(cdc, secret)
	if err != nil {
		return
	}

	var bz []byte
	genTx := GenTx{
		Name:    genTxConfig.Name,
		Address: addr,
		PubKey:  pk,
	}
	bz, err = wire.MarshalJSONIndent(cdc, genTx)
	if err != nil {
		return
	}
	appGenTx = json.RawMessage(bz)

	validator = tmtypes.GenesisValidator{
		PubKey: pk,
		// TODO: with the staking feature.
		Power: 1,
	}
	return
}

func makePrintMessage(cdc *wire.Codec, secret string) (json.RawMessage, error) {
	mm := map[string]string{"secret": secret}
	bz, err := cdc.MarshalJSON(mm)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(bz), nil
}

// AppGenState sets up the app_state and appends the cool app state
func BinanceAppGenState(cdc *wire.Codec, appGenTxs []json.RawMessage) (appState json.RawMessage, err error) {
	if len(appGenTxs) == 0 {
		err = errors.New("must provide at least 1 genesis transaction")
		return
	}

	genAccounts := make([]GenesisAccount, len(appGenTxs))
	for i, appGenTx := range appGenTxs {
		var genTx GenTx
		err = cdc.UnmarshalJSON(appGenTx, &genTx)
		if err != nil {
			return
		}

		// create the genesis account
		appAccount := types.AppAccount{BaseAccount: auth.NewBaseAccountWithAddress(genTx.Address)}
		if len(genTx.Name) > 0 {
			appAccount.SetName(genTx.Name)
		}
		acc := NewGenesisAccount(&appAccount)
		genAccounts[i] = acc
	}

	// create the final app state
	genesisState := GenesisState{
		Accounts:   genAccounts,
		Tokens:     append([]types.Token{}, tokens.DefaultGenesisToken(genAccounts[0].Address)),
		DexGenesis: dex.DefaultGenesis,
	}

	appState, err = wire.MarshalJSONIndent(cdc, genesisState)
	return
}
