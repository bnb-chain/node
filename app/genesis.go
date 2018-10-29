package app

import (
	"encoding/json"
	"errors"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/tendermint/tendermint/crypto"
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
		AppGenState: BinanceAppGenState,
	}
}

type GenTx struct {
	Name    string         `json:"name"`
	Address sdk.AccAddress `json:"address"`
	PubKey  crypto.PubKey  `json:"pub_key"`
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
