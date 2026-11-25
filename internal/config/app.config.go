package config

import (
	"time"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/servicetoken"
)

// Main holds the top-level application settings.
type Main struct {
	// Name of the application, as it appears in logs and tracing.
	Name string
	// DowntimeStart is when a planned downtime starts; nil when none is planned. From then until
	// it is removed, the service refuses work and leaves its database alone.
	DowntimeStart *time.Time
}

// Dependencies configures how the service reaches the services it calls.
type Dependencies struct {
	// ServiceJSONKeys locates the JSON Keys service, which verifies access tokens, and says how to
	// dial it.
	ServiceJSONKeys grpcf.ClientConfig
}

// Limits bounds what one request carries, below the REST server's own limits. The server's
// MaxRequestSize fits a batch, so the routes that write one item cap their body lower.
type Limits struct {
	// MaxItemRequestSize caps the body of a route that writes one item, in bytes.
	MaxItemRequestSize int64
}

// App is the complete configuration the service reads at startup.
type App struct {
	// App holds the top-level application settings.
	App Main
	// Dependencies configures the services the service calls.
	Dependencies Dependencies
	// Permissions maps each role to the permissions it grants.
	Permissions serviceauthentication.Permissions
	// Rest configures the REST server.
	Rest httpf.ServerConfig
	// ServiceToken names the service tokens the REST server accepts. Without an issuer, the server
	// refuses to start unless its IssuerOptional field lets it take every request.
	ServiceToken servicetoken.Config
	// Limits bounds what one request carries, below the REST server's own limits.
	Limits Limits

	// Otel configures the OpenTelemetry exporter.
	Otel otel.Config
	// Logger is the application logger the REST handlers report their errors to.
	Logger logging.Log
	// RestLogger configures the REST request logging middleware.
	RestLogger logging.HTTPConfig
	// Postgres configures the PostgreSQL connection.
	Postgres postgres.Config
}
