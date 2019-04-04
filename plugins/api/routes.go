package api

import (
	rpc "github.com/cosmos/cosmos-sdk/client/rpc"
	tx "github.com/cosmos/cosmos-sdk/client/tx"
	auth "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	bank "github.com/cosmos/cosmos-sdk/x/bank/client/rest"
	gov "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
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

	// auth routes
	r.HandleFunc(prefix+"/account/{address}", s.handleAccountReq(s.cdc, s.ctx)).
		Methods("GET")

	// tx routes
	r.HandleFunc(prefix+"/simulate", s.handleSimulateReq(s.cdc, s.ctx)).
		Methods("POST")

	// dex routes
	r.HandleFunc(prefix+"/markets", s.handlePairsReq(s.cdc, s.ctx)).
		Methods("GET")
	r.HandleFunc(prefix+"/depth", s.handleDexDepthReq(s.cdc, s.ctx)).
		Queries("symbol", "{symbol}", "limit", "{limit:[0-9]+}").
		Methods("GET")
	r.HandleFunc(prefix+"/depth", s.handleDexDepthReq(s.cdc, s.ctx)).
		Queries("symbol", "{symbol}").
		Methods("GET")
	r.HandleFunc(prefix+"/order", s.handleDexOrderReq(s.cdc, s.ctx, s.accStoreName)).
		Methods("PUT", "POST")

	r.HandleFunc(prefix+"/orders/open", s.handleDexOpenOrdersReq(s.cdc, s.ctx)).
		Queries("address", "{address}", "symbol", "{symbol}").
		Methods("GET")

	// tokens routes
	r.HandleFunc(prefix+"/tokens", s.handleTokensReq(s.cdc, s.ctx)).
		Methods("GET")
	r.HandleFunc(prefix+"/tokens/{symbol}", s.handleTokenReq(s.cdc, s.ctx)).
		Methods("GET")
	r.HandleFunc(prefix+"/balances/{address}", s.handleBalancesReq(s.cdc, s.ctx, s.tokens)).
		Methods("GET")
	r.HandleFunc(prefix+"/balances/{address}/{symbol}", s.handleBalanceReq(s.cdc, s.ctx, s.tokens)).
		Methods("GET")

	// fee params
	r.HandleFunc(prefix+"/fees", s.handleFeesParamReq(s.cdc, s.ctx)).
		Methods("GET")

	// stake query
	r.HandleFunc(prefix+"/stake/validators", s.handleValidatorsQueryReq(s.cdc, s.ctx)).
		Methods("GET")

	r.HandleFunc(prefix+"/stake/unbonding_delegations/delegator/{delegatorAddr}", s.handleDelegatorUnbondingDelegationsQueryReq(s.cdc, s.ctx)).
		Methods("GET")

	// legacy plugin routes
	// TODO: make these more like the above for simplicity.

	// keys rest routes disabled for security. while the nodes with keys (validators) run in a secure ringfenced environment,
	// disabling this is a precaution to protect third-party validators that might not have protected their networks adequately.
	//keys.RegisterRoutes(r, true)

	rpc.RegisterRoutes(s.ctx, r)
	tx.RegisterRoutes(s.ctx, r, s.cdc)
	auth.RegisterRoutes(s.ctx, r, s.cdc, s.accStoreName)
	bank.RegisterRoutes(s.ctx, r, s.cdc, s.keyBase)
	gov.RegisterRoutes(s.ctx, r, s.cdc)
	return s
}
