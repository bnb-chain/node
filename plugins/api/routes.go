package api

import (
	keys "github.com/cosmos/cosmos-sdk/client/keys"
	rpc "github.com/cosmos/cosmos-sdk/client/rpc"
	tx "github.com/cosmos/cosmos-sdk/client/tx"
	auth "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	bank "github.com/cosmos/cosmos-sdk/x/bank/client/rest"
)

const version = "v1"
const prefix = "/api/" + version

func (s *server) bindRoutes() *server {
	r := s.router

	// version routes
	r.HandleFunc("/version", s.handleVersion()).Methods("GET")
	r.HandleFunc("/node_version", s.handleNodeVersion()).Methods("GET")

	// dex routes
	r.HandleFunc(prefix+"/depth/{pair}", s.handleDexDepthRequest(s.cdc, s.ctx)).Methods("GET")

	// tokens routes
	r.HandleFunc(prefix+"/balances/{address}", s.handleBalancesRequest(s.cdc, s.ctx, s.tokens)).Methods("GET")
	r.HandleFunc(prefix+"/balances/{address}/{symbol}", s.handleBalanceRequest(s.cdc, s.ctx, s.tokens)).Methods("GET")

	// legacy plugin routes
	// TODO: make these more like the above for simplicity.
	keys.RegisterRoutes(r)
	rpc.RegisterRoutes(s.ctx, r)
	tx.RegisterRoutes(s.ctx, r, s.cdc)
	auth.RegisterRoutes(s.ctx, r, s.cdc, s.accStoreName)
	bank.RegisterRoutes(s.ctx, r, s.cdc, s.keyBase)

	return s
}
