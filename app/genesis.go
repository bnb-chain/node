package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cosmos/cosmos-sdk/server"
	serverCfg "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/stake"

	"github.com/tendermint/tendermint/crypto"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
)

const DefaultKeyPass = "12345678"

var (
	// each genesis validators will self delegate 1000e8 native tokens to become a validator
	DefaultSelfDelegationToken = sdk.NewCoin(types.NativeToken, 1000e8)
	// we put 20% of the total supply to the stake pool
	DefaultMaxBondedTokenAmount int64 = types.NativeTokenTotalSupply / 5
	// set default unbonding duration to 7 days
	DefaultUnbondingTime = 60 * 60 * 24 * 7 * time.Second
	// default max validators to 15
	DefaultMaxValidators uint16 = 15
)

type GenesisState struct {
	Tokens     []types.Token      `json:"tokens"`
	Accounts   []GenesisAccount   `json:"accounts"`
	DexGenesis dex.Genesis        `json:"dex"`
	StakeData  stake.GenesisState `json:"stake"`
	GenTxs     []json.RawMessage  `json:"gentxs"`
}

// GenesisAccount doesn't need pubkey or sequence
type GenesisAccount struct {
	Name    string         `json:"name"`
	Address sdk.AccAddress `json:"address"`
	ValAddr crypto.Address `json:"valaddr"`
}

// NewGenesisAccount -
func NewGenesisAccount(aa *types.AppAccount, valAddr crypto.Address) GenesisAccount {
	return GenesisAccount{
		Name:    aa.Name,
		Address: aa.GetAddress(),
		ValAddr: valAddr,
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

func BinanceAppGenTx(cdc *wire.Codec, valOperAddr sdk.ValAddress, pk crypto.PubKey, genTxConfig serverCfg.GenTx) (
	appGenTx, cliPrint json.RawMessage, validator tmtypes.GenesisValidator, err error) {

	// write app.toml when we run testnet command, we only know the `current` rootDir for each validator here
	// otherwise, we can only generate at ~/.bnbchaind/config/app.toml
	appConfigFilePath := filepath.Join(ServerContext.Context.Config.RootDir, "config/", config.AppConfigFileName+".toml")
	if _, err := os.Stat(appConfigFilePath); os.IsNotExist(err) {
		config.WriteConfigFile(appConfigFilePath, ServerContext.BinanceChainConfig)
	}
	return
}

// AppGenState sets up the app_state and appends the cool app state
func BinanceAppGenState(cdc *wire.Codec, appGenTxs []json.RawMessage) (appState json.RawMessage, err error) {
	if len(appGenTxs) == 0 {
		err = errors.New("must provide at least 1 genesis transaction")
		return
	}

	genAccounts := make([]GenesisAccount, len(appGenTxs))
	for i, genTx := range appGenTxs {
		var tx auth.StdTx
		if err = cdc.UnmarshalJSON(genTx, &tx); err != nil {
			return
		}
		msgs := tx.GetMsgs()
		if len(msgs) != 1 {
			err = errors.New(
				"must provide genesis StdTx with exactly 1 CreateValidator message")
			return
		}
		if msg, ok := msgs[0].(stake.MsgCreateValidator); !ok {
			err = fmt.Errorf(
				"genesis transaction %v does not contain a MsgCreateValidator", i)
			return
		} else {
			appAccount := types.AppAccount{BaseAccount: auth.NewBaseAccountWithAddress(sdk.AccAddress(msg.ValidatorAddr))}
			if len(msg.Moniker) > 0 {
				appAccount.SetName(msg.Moniker)
			}
			acc := NewGenesisAccount(&appAccount, msg.PubKey.Address())
			genAccounts[i] = acc
		}
	}

	stakeData := stake.DefaultGenesisState()
	nativeToken := tokens.DefaultGenesisToken(genAccounts[0].Address)
	stakeData.Pool.LooseTokens = sdk.NewDec(DefaultMaxBondedTokenAmount)
	stakeData.Params.BondDenom = nativeToken.Symbol
	stakeData.Params.UnbondingTime = DefaultUnbondingTime
	stakeData.Params.MaxValidators = DefaultMaxValidators
	genesisState := GenesisState{
		Accounts:   genAccounts,
		Tokens:     []types.Token{nativeToken},
		DexGenesis: dex.DefaultGenesis,
		StakeData:  stakeData,
		GenTxs:     appGenTxs,
	}

	appState, err = wire.MarshalJSONIndent(cdc, genesisState)
	return
}
