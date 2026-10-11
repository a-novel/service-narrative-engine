package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// RestLimit caps each visitor at limit requests per window on each route, then answers 429 with
// Retry-After. Install it on a router group: the route pattern is only known once routing is done.
func RestLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	return httprate.LimitBy(limit, window, httprate.JoinKeys(restVisitorKey, restRouteKey))
}

// restVisitorKey counts an IPv6 visitor per /64, the block one client usually controls.
func restVisitorKey(r *http.Request) (string, error) {
	return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
}

// restRouteKey gives each route its own budget, shared by all its path parameters.
func restRouteKey(r *http.Request) (string, error) {
	return r.Method + " " + chi.RouteContext(r.Context()).RoutePattern(), nil
}
