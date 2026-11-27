package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-narrative-engine/v2/internal/config"
)

// RestLimits are the limits on routes that cost more than the router's standard rate covers. Mount each
// after the auth middleware, with chi's With, so a signed-in caller counts as its user.
type RestLimits struct {
	// DocumentWrite caps each caller's writes of a document: the content of an instance, or the links of a
	// project. It names what a route writes, so every route that writes a document mounts it, and each counts
	// on a budget of its own.
	DocumentWrite func(http.Handler) http.Handler
	// PluginVersionWrite caps the versions each API key publishes, within a short window and a long one. Mount
	// it after the publisher gate, which verifies the key of the request: a request the gate refuses spends
	// nothing of a key's budget. A request no key signed, which only a local stack takes, passes uncounted.
	PluginVersionWrite func(http.Handler) http.Handler
}

// NewRestLimits builds the configured policies. The REST config may turn the rates off for test stacks.
func NewRestLimits(rest httpf.ServerConfig, policies config.Limits) RestLimits {
	limits := httpf.NewLimits(rest)

	documentWrite := limits.By(policies.RateLimitDocumentWrite, serviceauthentication.CallerKey)

	versionWriteBurst := limits.By(policies.RateLimitPluginVersionWriteBurst, apiKeyKey)
	versionWrite := limits.By(policies.RateLimitPluginVersionWrite, apiKeyKey)
	pluginVersionWrite := middleware.Maybe(chi.Chain(versionWriteBurst, versionWrite).Handler, signedByAPIKey)

	return RestLimits{DocumentWrite: documentWrite, PluginVersionWrite: pluginVersionWrite}
}

// signedByAPIKey reports whether an API key signed the request, as the publisher gate verified.
func signedByAPIKey(r *http.Request) bool {
	_, signed := servicejsonkeys.RequestAPIKey(r.Context())

	return signed
}

// apiKeyKey keys a limit by the API key that signed the request, as the publisher gate verified.
func apiKeyKey(r *http.Request) (string, error) {
	keyID, _ := servicejsonkeys.RequestAPIKey(r.Context())

	return "apikey:" + keyID, nil
}
