package app

import (
	"encoding/json"

	"github.com/BiJie/BinanceChain/common"
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
	tokenMapper   tokens.Mapper

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

	app.tokenMapper = tokens.NewTokenMapper(cdc, common.TokenStoreKey)

	// Add handlers.
	coinKeeper := bank.NewCoinKeeper(app.accountMapper)
	dexKeeper := dex.NewKeeper(common.MainStoreKey, coinKeeper)
	ibcMapper := ibc.NewIBCMapper(app.cdc, common.IBCStoreKey)
	stakeKeeper := simplestake.NewKeeper(common.StakingStoreKey, coinKeeper)
	app.Router().
		AddRoute("bank", bank.NewHandler(coinKeeper)).
		AddRoute("tokens", tokens.NewHandler(app.tokenMapper, coinKeeper)).
		AddRoute("dex", dex.NewHandler(dexKeeper)).
		AddRoute("ibc", ibc.NewHandler(ibcMapper, coinKeeper)).
		AddRoute("simplestake", simplestake.NewHandler(stakeKeeper))

	// Define the feeHandler.
	app.feeHandler = auth.BurnFeeHandler

	// Initialize BaseApp.
	app.SetTxDecoder(app.txDecoder)
	app.SetInitChainer(app.initChainerFn(dexKeeper))
	app.MountStoresIAVL(common.MainStoreKey, common.AccountStoreKey, common.TokenStoreKey, common.IBCStoreKey, common.StakingStoreKey)
	app.SetAnteHandler(auth.NewAnteHandler(app.accountMapper, app.feeHandler))
	err := app.LoadLatestVersion(common.MainStoreKey)
	if err != nil {
		cmn.Exit(err.Error())
	}

	return app
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
func (app *BasecoinApp) initChainerFn(dexKeeper dex.Keeper) sdk.InitChainer {
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
		err = dexKeeper.InitGenesis(ctx, genesisState.DexGenesis)
		if err != nil {
			panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
			//	return sdk.ErrGenesisParse("").TraceCause(err, "")
		}

		return abci.ResponseInitChain{}
	}
}
