package api

import (
	"net/http"

	hnd "github.com/BiJie/BinanceChain/plugins/api/handlers"
)

func (s *server) handleVersion() http.HandlerFunc {
	return hnd.CLIVersionRequestHandler
}

func (s *server) handleNodeVersion() http.HandlerFunc {
	return hnd.NodeVersionRequestHandler(s.ctx)
}
