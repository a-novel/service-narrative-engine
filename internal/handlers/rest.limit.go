package handlers

import (
	"net/http"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-narrative-engine/v2/internal/config"
)

// RestLimits are the limits on routes that cost more than the router's standard rate covers. Mount each
// after the auth middleware, with chi's With, so a signed-in caller counts as its user.
type RestLimits struct {
	// DocumentWrite caps each caller's writes of a document: the content of an instance, or its links. It
	// names what a route writes, so every route that writes a document mounts it, and each counts on a budget
	// of its own.
	DocumentWrite func(http.Handler) http.Handler
}

// NewRestLimits builds the configured policies. The REST config may turn the rates off for test stacks.
func NewRestLimits(rest httpf.ServerConfig, policies config.Limits) RestLimits {
	limits := httpf.NewLimits(rest)

	documentWrite := limits.By(policies.RateLimitDocumentWrite, serviceauthentication.CallerKey)

	return RestLimits{DocumentWrite: documentWrite}
}
