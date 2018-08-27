package api

import (
	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
	keyscli "github.com/cosmos/cosmos-sdk/client/keys"
	keys "github.com/cosmos/cosmos-sdk/crypto/keys"

	"github.com/BiJie/BinanceChain/common"
	tkstore "github.com/BiJie/BinanceChain/plugins/tokens/store"
	"github.com/BiJie/BinanceChain/wire"
)

const maxPostSize int64 = 1024 * 1024 * 0.5 // ~500KB

type server struct {
	router *mux.Router

	// settings
	maxPostSize int64

	// handler dependencies
	ctx context.CoreContext
	cdc *wire.Codec

	// stores for handlers
	keyBase keys.Keybase
	tokens  tkstore.Mapper

	accStoreName string
}

// NewServer provides a new server structure.
func newServer(ctx context.CoreContext, cdc *wire.Codec) *server {
	kb, err := keyscli.GetKeyBase()
	if err != nil {
		panic(err)
	}

	return &server{
		router:       mux.NewRouter(),
		maxPostSize:  maxPostSize,
		ctx:          ctx,
		cdc:          cdc,
		keyBase:      kb,
		tokens:       tkstore.NewMapper(cdc, common.TokenStoreKey),
		accStoreName: common.AccountStoreName,
	}
}
