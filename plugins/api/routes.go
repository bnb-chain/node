package api

import (
	keys "github.com/cosmos/cosmos-sdk/client/keys"
	rpc "github.com/cosmos/cosmos-sdk/client/rpc"
	tx "github.com/cosmos/cosmos-sdk/client/tx"
	auth "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	bank "github.com/cosmos/cosmos-sdk/x/bank/client/rest"

	"github.com/BiJie/BinanceChain/common"
	dex "github.com/BiJie/BinanceChain/plugins/dex/client/rest"
	tokens "github.com/BiJie/BinanceChain/plugins/tokens/client/rest"
	tkstore "github.com/BiJie/BinanceChain/plugins/tokens/store"
)

const version = "v1"
const prefix = "/api/" + version

func (s *server) bindRoutes() *server {
	r := s.router

	// version routes
	r.HandleFunc("/version", s.handleVersion()).Methods("GET")
	r.HandleFunc("/node_version", s.handleNodeVersion()).Methods("GET")

	// dex routes
	r.HandleFunc(prefix+"/depth/{pair}", dex.DepthRequestHandler(s.cdc, s.ctx)).Methods("GET")

	// legacy plugin routes
	// TODO: make these more like the above for simplicity.
	keys.RegisterRoutes(r)
	rpc.RegisterRoutes(s.ctx, r)
	tx.RegisterRoutes(s.ctx, r, s.cdc)
	auth.RegisterRoutes(s.ctx, r, s.cdc, s.accountStoreName)
	bank.RegisterRoutes(s.ctx, r, s.cdc, s.keyBase)
	tokens.RegisterRoutes(s.ctx, r, s.cdc, tkstore.NewMapper(cdc, common.TokenStoreKey))

	return s
}
