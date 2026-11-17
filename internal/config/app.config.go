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
	// ServiceJSONKeysHost is the hostname of the JSON Keys service, which verifies access tokens.
	ServiceJSONKeysHost string
	// ServiceJSONKeysPort is the gRPC port of the JSON Keys service.
	ServiceJSONKeysPort int
	// ServiceJSONKeysCredentials authenticates the calls to the JSON Keys service.
	ServiceJSONKeysCredentials grpcf.CredentialsProvider

	// ServiceGenaiHost is the hostname of the genai service, which runs the model calls of generations.
	// Empty leaves genai unconfigured.
	ServiceGenaiHost string
	// ServiceGenaiPort is the gRPC port of the genai service.
	ServiceGenaiPort int
	// ServiceGenaiCredentials authenticates the calls to the genai service.
	ServiceGenaiCredentials grpcf.CredentialsProvider
}

// Limits bounds what one request carries, below the REST server's own limits. The server's
// MaxRequestSize fits a batch, so the routes that write one item cap their body lower.
type Limits struct {
	// MaxItemRequestSize caps the body of a route that writes one item, in bytes.
	MaxItemRequestSize int64
	// MaxStreamDuration bounds a streamed response, past the server's request timeout.
	MaxStreamDuration time.Duration
	// MaxStreamsPerCaller caps the streams one caller holds open on an instance, across the stream
	// routes. A stream past it answers 429.
	MaxStreamsPerCaller int
	// RateLimitGeneration caps each signed-in user's generation submits, on top of the router's rate. A submit
	// past it answers 429.
	RateLimitGeneration httpf.Rate
	// RateLimitGenerationRead caps each signed-in user's reads of a call, on top of the router's rate. A read and a
	// stream each count on a budget of their own. A read past it answers 429.
	RateLimitGenerationRead httpf.Rate
	// MaxConcurrentProjectReads caps the reads of a whole project an instance runs at once: manuscript exports
	// and the snapshots generations plan from. A read past it waits for a slot. 0 sizes it to the instance's
	// memory.
	MaxConcurrentProjectReads int
}

// App is the complete configuration the service reads at startup.
type App struct {
	// App holds the top-level application settings.
	App Main
	// Dependencies configures the services the service calls.
	Dependencies Dependencies
	// Permissions maps each role to the permissions it grants.
	Permissions serviceauthentication.Permissions
	// Caps maps roles to the caps genai counts their users' generations against.
	Caps Caps
	// Rest configures the REST server.
	Rest httpf.ServerConfig
	// ServiceToken names the service tokens the REST server accepts. Without an issuer, the server
	// refuses to start unless its IssuerOptional field lets it take every request.
	ServiceToken servicetoken.Config
	// Limits bounds what one request carries, below the REST server's own limits.
	Limits Limits
	// Generations configures the model calls of generations.
	Generations Generations

	// Otel configures the OpenTelemetry exporter.
	Otel otel.Config
	// Logger is the application logger the REST handlers report their errors to.
	Logger logging.Log
	// RestLogger configures the REST request logging middleware.
	RestLogger logging.HTTPConfig
	// Postgres configures the PostgreSQL connection.
	Postgres postgres.Config
}
