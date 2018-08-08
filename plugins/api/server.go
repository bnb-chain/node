package api

import (
	"github.com/cosmos/cosmos-sdk/client/context"
	keyscli "github.com/cosmos/cosmos-sdk/client/keys"
	keys "github.com/cosmos/cosmos-sdk/crypto/keys"
	"github.com/gorilla/mux"

	"github.com/BiJie/BinanceChain/common"
	tkstore "github.com/BiJie/BinanceChain/plugins/tokens/store"
	"github.com/BiJie/BinanceChain/wire"
)

type server struct {
	router *mux.Router

	// handler dependencies
	ctx context.CoreContext
	cdc *wire.Codec

	// stores for handlers
	keyBase          keys.Keybase
	tokenMapper      tkstore.Mapper
	accountStoreName string
}

// NewServer provides a new server structure.
func newServer(ctx context.CoreContext, cdc *wire.Codec) *server {
	kb, err := keyscli.GetKeyBase()
	if err != nil {
		panic(err)
	}

	return &server{
		router:           mux.NewRouter(),
		ctx:              ctx,
		cdc:              cdc,
		keyBase:          kb,
		tokenMapper:      tkstore.NewMapper(cdc, common.TokenStoreKey),
		accountStoreName: common.AccountStoreName,
	}
}
