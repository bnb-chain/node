package app

import (
	"encoding/json"

	"github.com/BiJie/BinanceChain/common"
	tokenStore "github.com/BiJie/BinanceChain/plugins/tokens/store"
	abci "github.com/tendermint/abci/types"
	cmn "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"
	"github.com/tendermint/tmlibs/log"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/simplestake"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/tokens"
)

const (
	appName = "BNBChain"
)

// Extended ABCI application
type BasecoinApp struct {
	*bam.BaseApp
	cdc *wire.Codec

	// Manage getting and setting accounts
	accountMapper sdk.AccountMapper
	tokenMapper   tokenStore.Mapper
	coinKeeper    bank.CoinKeeper
	dexKeeper     dex.Keeper

	// Handle fees
	feeHandler sdk.FeeHandler
}

func NewBasecoinApp(logger log.Logger, db dbm.DB) *BasecoinApp {

	// Create app-level codec for txs and accounts.
	var cdc = MakeCodec()

	// Create your application object.
	var app = &BasecoinApp{
		BaseApp: bam.NewBaseApp(appName, logger, db),
		cdc:     cdc,
	}

	// Define the accountMapper.
	app.accountMapper = auth.NewAccountMapper(
		cdc,
		common.AccountStoreKey, // target store
		&types.AppAccount{},    // prototype
	).Seal()

	app.tokenMapper = tokenStore.NewMapper(cdc, common.TokenStoreKey)

	// Add handlers.
	app.coinKeeper = bank.NewCoinKeeper(app.accountMapper)
	app.dexKeeper = dex.NewKeeper(common.MainStoreKey, app.coinKeeper)
	// Currently we do not need the ibc and staking part
	// ibcMapper := ibc.NewIBCMapper(app.cdc, common.IBCStoreKey)
	// stakeKeeper := simplestake.NewKeeper(common.StakingStoreKey, coinKeeper)
	app.registerHandlers()

	// Define the feeHandler.
	app.feeHandler = auth.BurnFeeHandler

	// Initialize BaseApp.
	app.SetTxDecoder(app.txDecoder)
	app.SetInitChainer(app.initChainerFn())
	app.MountStoresIAVL(common.MainStoreKey, common.AccountStoreKey, common.TokenStoreKey, common.IBCStoreKey, common.StakingStoreKey)
	app.SetAnteHandler(auth.NewAnteHandler(app.accountMapper, app.feeHandler))
	err := app.LoadLatestVersion(common.MainStoreKey)
	if err != nil {
		cmn.Exit(err.Error())
	}

	return app
}

func (app *BasecoinApp) registerHandlers() {
	app.Router().
		AddRoute("bank", bank.NewHandler(app.coinKeeper)).
		AddRoute("dex", dex.NewHandler(app.dexKeeper))
	// AddRoute("ibc", ibc.NewHandler(ibcMapper, coinKeeper)).
	// AddRoute("simplestake", simplestake.NewHandler(stakeKeeper))
	for route, handler := range tokens.Routes(app.tokenMapper, app.accountMapper, app.coinKeeper) {
		app.Router().AddRoute(route, handler)
	}
}

// Custom tx codec
func MakeCodec() *wire.Codec {
	var cdc = wire.NewCodec()

	// Register Msgs
	cdc.RegisterInterface((*sdk.Msg)(nil), nil)
	cdc.RegisterConcrete(bank.SendMsg{}, "basecoin/Send", nil)

	cdc.RegisterConcrete(dex.MakeOfferMsg{}, "dex/MakeOfferMsg", nil)
	cdc.RegisterConcrete(dex.FillOfferMsg{}, "dex/FillOfferMsg", nil)
	cdc.RegisterConcrete(dex.CancelOfferMsg{}, "dex/CancelOfferMsg", nil)
	cdc.RegisterConcrete(ibc.IBCTransferMsg{}, "basecoin/IBCTransferMsg", nil)
	cdc.RegisterConcrete(ibc.IBCReceiveMsg{}, "basecoin/IBCReceiveMsg", nil)
	cdc.RegisterConcrete(simplestake.BondMsg{}, "basecoin/BondMsg", nil)
	cdc.RegisterConcrete(simplestake.UnbondMsg{}, "basecoin/UnbondMsg", nil)

	types.RegisterTypes(cdc)
	tokens.RegisterTypes(cdc)
	// Register crypto.
	wire.RegisterCrypto(cdc)

	return cdc
}

// Custom logic for transaction decoding
func (app *BasecoinApp) txDecoder(txBytes []byte) (sdk.Tx, sdk.Error) {
	var tx = sdk.StdTx{}

	if len(txBytes) == 0 {
		return nil, sdk.ErrTxDecode("txBytes are empty")
	}

	// StdTx.Msg is an interface. The concrete types
	// are registered by MakeTxCodec in bank.RegisterAmino.
	err := app.cdc.UnmarshalBinary(txBytes, &tx)
	if err != nil {
		return nil, sdk.ErrTxDecode("").TraceCause(err, "")
	}
	return tx, nil
}

// custom logic for democoin initialization
func (app *BasecoinApp) initChainerFn() sdk.InitChainer {
	return func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		stateJSON := req.AppStateBytes

		genesisState := new(types.GenesisState)
		err := json.Unmarshal(stateJSON, genesisState)
		if err != nil {
			panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
			// return sdk.ErrGenesisParse("").TraceCause(err, "")
		}

		for _, gacc := range genesisState.Accounts {
			acc, err := gacc.ToAppAccount()
			if err != nil {
				panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
				//	return sdk.ErrGenesisParse("").TraceCause(err, "")
			}
			app.accountMapper.SetAccount(ctx, acc)
		}

		// Application specific genesis handling
		err = app.dexKeeper.InitGenesis(ctx, genesisState.DexGenesis)
		if err != nil {
			panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
			//	return sdk.ErrGenesisParse("").TraceCause(err, "")
		}

		return abci.ResponseInitChain{}
	}
}
