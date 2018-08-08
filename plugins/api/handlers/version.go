package handlers

import (
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/version"
)

// CLIVersionRequestHandler handles requests to the cli version REST handler endpoint
func CLIVersionRequestHandler(w http.ResponseWriter, r *http.Request) {
	v := version.GetVersion()
	w.Write([]byte(v))
}

// NodeVersionRequestHandler handles requests to the connected node version REST handler endpoint
func NodeVersionRequestHandler(ctx context.CoreContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version, err := ctx.Query("/app/version")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Could't query version. Error: %s", err.Error())))
			return
		}
		w.Write(version)
	}
}
