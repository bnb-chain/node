package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/tendermint/tendermint/crypto"

	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/dex"
	"github.com/binance-chain/node/plugins/param"
	paramtypes "github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/wire"
)

const DefaultKeyPass = "12345678"

var (
	// each genesis validators will self delegate 1000e8 native tokens to become a validator
	DefaultSelfDelegationToken = sdk.NewCoin(types.NativeTokenSymbol, 1000e8)
	// we put 20% of the total supply to the stake pool
	DefaultMaxBondedTokenAmount int64 = types.NativeTokenTotalSupply / 5
	// set default unbonding duration to 7 days
	DefaultUnbondingTime = 60 * 60 * 24 * 7 * time.Second
	// default max validators to 15
	DefaultMaxValidators uint16 = 15

	// min gov deposit
	DefaultGovMinDesposit = sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 2000e8)}
)

type GenesisState struct {
	Tokens       []tokens.GenesisToken   `json:"tokens"`
	Accounts     []GenesisAccount        `json:"accounts"`
	DexGenesis   dex.Genesis             `json:"dex"`
	ParamGenesis paramtypes.GenesisState `json:"param"`
	StakeData    stake.GenesisState      `json:"stake"`
	GovData      gov.GenesisState        `json:"gov"`
	GenTxs       []json.RawMessage       `json:"gentxs"`
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
		if msg, ok := msgs[0].(stake.MsgCreateValidatorProposal); !ok {
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

	govData := gov.DefaultGenesisState()
	govData.DepositProcedure.MinDeposit = DefaultGovMinDesposit

	genesisState := GenesisState{
		Tokens:       []tokens.GenesisToken{nativeToken},
		Accounts:     genAccounts,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
		StakeData:    stakeData,
		GenTxs:       appGenTxs,
		GovData:      govData,
	}

	appState, err = wire.MarshalJSONIndent(cdc, genesisState)
	return
}
