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
	r.HandleFunc("/version", s.handleVersionReq()).
		Methods("GET")
	r.HandleFunc("/node_version", s.handleNodeVersionReq()).
		Methods("GET")

	// dex routes
	r.HandleFunc(prefix+"/pairs", s.handlePairsReq(s.cdc, s.ctx)).
		Methods("GET")
	r.HandleFunc(prefix+"/depth", s.handleDexDepthReq(s.cdc, s.ctx)).
		Queries("symbol", "{symbol}", "limit", "{limit:[0-9]+}").
		Methods("GET")
	r.HandleFunc(prefix+"/depth", s.handleDexDepthReq(s.cdc, s.ctx)).
		Queries("symbol", "{symbol}").
		Methods("GET")
	r.HandleFunc(prefix+"/order", s.handleDexOrderReq(s.cdc, s.ctx, s.accStoreName)).
		Methods("PUT", "POST")

	// tokens routes
	r.HandleFunc(prefix+"/tokens", s.handleTokensReq(s.cdc, s.ctx)).
		Methods("GET")
	r.HandleFunc(prefix+"/balances/{address}", s.handleBalancesReq(s.cdc, s.ctx, s.tokens)).
		Methods("GET")
	r.HandleFunc(prefix+"/balances/{address}/{symbol}", s.handleBalanceReq(s.cdc, s.ctx, s.tokens)).
		Methods("GET")

	// legacy plugin routes
	// TODO: make these more like the above for simplicity.
	keys.RegisterRoutes(r)
	rpc.RegisterRoutes(s.ctx, r)
	tx.RegisterRoutes(s.ctx, r, s.cdc)
	auth.RegisterRoutes(s.ctx, r, s.cdc, s.accStoreName)
	bank.RegisterRoutes(s.ctx, r, s.cdc, s.keyBase)

	return s
}
