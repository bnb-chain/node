package router

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Router provides handlers for each transaction type.
type Router interface {
	AddRoute(r string, h Handler) (rtr Router)
	Route(path string) (h Handler)
}

// map a transaction type to a handler and an initgenesis function
type route struct {
	r string
	h Handler
}

type router struct {
	routes []route
}

// Handler defines the core of the state transition function of an application.
type Handler func(ctx sdk.Context, msg sdk.Msg, simulate bool) sdk.Result
