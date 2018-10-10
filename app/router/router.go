package router

import (
	"regexp"
)

// NewRouter creates a new router
// TODO either make Function unexported or make return type (router) Exported
func NewRouter() *router {
	return &router{
		routes: make([]route, 0),
	}
}

var isAlpha = regexp.MustCompile(`^[a-zA-Z]+$`).MatchString

// AddRoute adds a msg route to the router.
func (rtr *router) AddRoute(r string, h Handler) Router {
	if !isAlpha(r) {
		panic("route expressions can only contain alphabet characters")
	}
	rtr.routes = append(rtr.routes, route{r, h})

	return rtr
}

// Route triggers the routing logic of the Router.
// TODO handle expressive matches.
func (rtr *router) Route(path string) (h Handler) {
	for _, route := range rtr.routes {
		if route.r == path {
			return route.h
		}
	}
	return nil
}
