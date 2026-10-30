package config

import (
	"time"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/sentryf"
	"github.com/a-novel-kit/golib/servicetoken"
)

// Main holds the top-level application settings.
type Main struct {
	// Name of the application, as it appears in logs and tracing.
	Name string `json:"name" yaml:"name"`
	// DowntimeStart is when a planned downtime starts; nil when none is planned. From then until
	// it is removed, the service refuses work and leaves its database alone.
	DowntimeStart *time.Time `json:"downtimeStart" yaml:"downtimeStart"`
}

// Dependencies configures how the service reaches the services it calls.
type Dependencies struct {
	// ServiceJsonKeysHost is the hostname of the JSON Keys service, which verifies access tokens.
	ServiceJsonKeysHost string `json:"serviceJSONKeysHost" yaml:"serviceJSONKeysHost"`
	// ServiceJsonKeysPort is the gRPC port of the JSON Keys service.
	ServiceJsonKeysPort int `json:"serviceJSONKeysPort" yaml:"serviceJSONKeysPort"`
	// ServiceJsonKeysCredentials authenticates the calls to the JSON Keys service.
	ServiceJsonKeysCredentials grpcf.CredentialsProvider `json:"-" yaml:"-"`

	// ServiceGenaiHost is the hostname of the genai service, which runs the model calls of generations.
	// Empty leaves genai unconfigured.
	ServiceGenaiHost string `json:"serviceGenaiHost" yaml:"serviceGenaiHost"`
	// ServiceGenaiPort is the gRPC port of the genai service.
	ServiceGenaiPort int `json:"serviceGenaiPort" yaml:"serviceGenaiPort"`
	// ServiceGenaiCredentials authenticates the calls to the genai service.
	ServiceGenaiCredentials grpcf.CredentialsProvider `json:"-" yaml:"-"`
}

// Limits bounds what one request carries, below the REST server's own limits. The server's
// MaxRequestSize fits a batch, so the routes that write one item cap their body lower.
type Limits struct {
	// ItemMaxRequestSize caps the body of a route that writes one item, in bytes.
	ItemMaxRequestSize int64 `json:"itemMaxRequestSize" yaml:"itemMaxRequestSize"`
}

// App is the complete configuration the service reads at startup.
type App struct {
	// App holds the top-level application settings.
	App Main `json:"app" yaml:"app"`
	// Dependencies configures the services the service calls.
	Dependencies Dependencies `json:"dependencies" yaml:"dependencies"`
	// Permissions maps each role to the permissions it grants.
	Permissions serviceauthentication.Permissions `json:"permissions" yaml:"permissions"`
	// Rest configures the REST server.
	Rest httpf.ServerConfig `json:"rest" yaml:"rest"`
	// ServiceToken names the service tokens the REST server accepts. Without an issuer, it takes every
	// request without one.
	ServiceToken servicetoken.Config `json:"serviceToken" yaml:"serviceToken"`
	// Limits bounds what one request carries, below the REST server's own limits.
	Limits Limits `json:"limits" yaml:"limits"`
	// Generations configures the model calls of generations.
	Generations Generations `json:"generations" yaml:"generations"`

	// Otel configures the OpenTelemetry exporter.
	Otel otel.Config `json:"otel" yaml:"otel"`
	// Sentry configures the reporter that sends server faults to Sentry.
	Sentry sentryf.Config `json:"sentry" yaml:"sentry"`
	// Logger is the application logger the REST handlers report their errors to.
	Logger logging.Log `json:"logger" yaml:"logger"`
	// RestLogger configures the REST request logging middleware.
	RestLogger logging.HTTPConfig `json:"restLogger" yaml:"restLogger"`
	// Postgres configures the PostgreSQL connection.
	Postgres postgres.Config `json:"postgres" yaml:"postgres"`
}
