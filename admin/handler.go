package admin

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/config"

	"github.com/binance-chain/node/common/runtime"
	"github.com/binance-chain/node/common/types"
)

// path:
// set to some mode: admin/mode/{mode}/{nonce}, nonce is a random number used together with req.Data to verify the priv key
// get current mode: admin/mode/{nonce}
func GetHandler(config *config.Config) types.AbciQueryHandler {
	return func(appp types.ChainApp, req abci.RequestQuery, path []string) *abci.ResponseQuery {
		if (len(path) != 3 && len(path) != 4) || path[0] != "admin" || path[1] != "mode" {
			result := sdk.ErrUnknownRequest(req.Path).QueryResult()
			return &result
		}

		pvFile := config.PrivValidatorKeyFile()
		_, pubKey, err := readPrivValidator(pvFile)
		if err != nil {
			result := sdk.ErrInternal(err.Error()).QueryResult()
			return &result
		}

		if len(path) == 3 {
			nonce := path[2]
			if !pubKey.VerifyBytes([]byte(nonce), req.Data) {
				res := sdk.ErrUnauthorized("permission denied").QueryResult()
				return &res
			}
			res := abci.ResponseQuery{
				Code:  uint32(sdk.ABCICodeOK),
				Value: []byte{uint8(runtime.RunningMode)},
			}
			return &res
		}

		// len == 4
		mode := path[2]
		nonce := path[3]
		if !pubKey.VerifyBytes([]byte(nonce), req.Data) {
			res := sdk.ErrUnauthorized("permission denied").QueryResult()
			return &res
		}

		if mode == "0" {
			runtime.RunningMode = runtime.NormalMode
		} else if mode == "1" {
			runtime.RunningMode = runtime.TransferOnlyMode
		} else if mode == "2" {
			runtime.RunningMode = runtime.RecoverOnlyMode
		} else {
			res := sdk.ErrUnknownRequest("invalid mode").QueryResult()
			return &res
		}

		res := abci.ResponseQuery{
			Code:  uint32(sdk.ABCICodeOK),
			Value: []byte{uint8(runtime.RunningMode)},
		}
		return &res
	}
}
