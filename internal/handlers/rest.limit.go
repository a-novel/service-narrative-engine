package handlers

import (
	"net/http"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

// RestLimits are the limits on routes that cost more than the router's standard rate covers. Mount each
// after the auth middleware, with chi's With, so a signed-in caller counts as its user.
type RestLimits struct {
	// Stream caps the streams each caller holds open, across both stream routes.
	Stream func(http.Handler) http.Handler
	// Generation caps each caller's generation submits, which call a paid provider.
	Generation func(http.Handler) http.Handler
}

// NewRestLimits builds the configured policies. The REST config may turn the rates off for test stacks;
// the concurrency gates stay.
func NewRestLimits(rest httpf.ServerConfig, policies config.Limits) RestLimits {
	limits := httpf.NewLimits(rest)

	streams := httpf.LimitConcurrencyBy(policies.MaxStreamsPerCaller, serviceauthentication.CallerKey)
	generation := limits.By(policies.GenerationRate, serviceauthentication.CallerKey)

	return RestLimits{Stream: streams, Generation: generation}
}
