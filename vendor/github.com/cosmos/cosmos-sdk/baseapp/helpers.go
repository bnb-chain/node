package baseapp

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
)

// nolint - Mostly for testing
func (app *BaseApp) Check(tx sdk.Tx) (result sdk.Result) {
	txHash := cmn.HexBytes(tmhash.Sum(nil)).String()
	return app.RunTx(sdk.RunTxModeCheck, tx, txHash)
}

// nolint - full tx execution
func (app *BaseApp) Simulate(txBytes []byte, tx sdk.Tx) (result sdk.Result) {
	txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
	return app.RunTx(sdk.RunTxModeSimulate, tx, txHash)
}

// nolint
func (app *BaseApp) Deliver(tx sdk.Tx) (result sdk.Result) {
	txHash := cmn.HexBytes(tmhash.Sum(nil)).String()
	return app.RunTx(sdk.RunTxModeDeliver, tx, txHash)
}
