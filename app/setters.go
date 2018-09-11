package app

import (
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	dbm "github.com/tendermint/tendermint/libs/db"

	"github.com/BiJie/BinanceChain/common/types"
)

// nolint - Setter functions
func (app *BaseApp) SetName(name string) {
	if app.sealed {
		panic("SetName() on sealed BaseApp")
	}
	app.name = name
}
func (app *BaseApp) SetDB(db dbm.DB) {
	if app.sealed {
		panic("SetDB() on sealed BaseApp")
	}
	app.db = db
}
func (app *BaseApp) SetCMS(cms store.CommitMultiStore) {
	if app.sealed {
		panic("SetEndBlocker() on sealed BaseApp")
	}
	app.cms = cms
}
func (app *BaseApp) SetTxDecoder(txDecoder sdk.TxDecoder) {
	if app.sealed {
		panic("SetTxDecoder() on sealed BaseApp")
	}
	app.txDecoder = txDecoder
}
func (app *BaseApp) SetInitChainer(initChainer types.InitChainer) {
	if app.sealed {
		panic("SetInitChainer() on sealed BaseApp")
	}
	app.initChainer = initChainer
}
func (app *BaseApp) SetBeginBlocker(beginBlocker types.BeginBlocker) {
	if app.sealed {
		panic("SetBeginBlocker() on sealed BaseApp")
	}
	app.beginBlocker = beginBlocker
}
func (app *BaseApp) SetEndBlocker(endBlocker types.EndBlocker) {
	if app.sealed {
		panic("SetEndBlocker() on sealed BaseApp")
	}
	app.endBlocker = endBlocker
}
func (app *BaseApp) SetAnteHandler(ah types.AnteHandler) {
	if app.sealed {
		panic("SetAnteHandler() on sealed BaseApp")
	}
	app.anteHandler = ah
}
func (app *BaseApp) SetAddrPeerFilter(pf types.PeerFilter) {
	if app.sealed {
		panic("SetAddrPeerFilter() on sealed BaseApp")
	}
	app.addrPeerFilter = pf
}
func (app *BaseApp) SetPubKeyPeerFilter(pf types.PeerFilter) {
	if app.sealed {
		panic("SetPubKeyPeerFilter() on sealed BaseApp")
	}
	app.pubkeyPeerFilter = pf
}
func (app *BaseApp) Router() Router {
	if app.sealed {
		panic("Router() on sealed BaseApp")
	}
	return app.router
}
func (app *BaseApp) Seal()          { app.sealed = true }
func (app *BaseApp) IsSealed() bool { return app.sealed }
func (app *BaseApp) enforceSeal() {
	if !app.sealed {
		panic("enforceSeal() on BaseApp but not sealed")
	}
}
