package env

import (
	"os"
	"time"

	"github.com/a-novel-kit/golib/config"
	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/httpf"
)

// prefix is prepended to every configuration variable name that this package reads.
// Set SERVICE_NARRATIVE_ENGINE_ENV_PREFIX when embedding the service in another project,
// where unprefixed variable names could collide with the host project's own.
var prefix = os.Getenv("SERVICE_NARRATIVE_ENGINE_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

const (
	// AppNameDefault names the application in logs and traces when APP_NAME is unset.
	AppNameDefault = "service-narrative-engine"

	// TimeoutShutdownDefault ends a server's drain 6 seconds after SIGTERM, so its OpenTelemetry
	// flush, up to 2 seconds, finishes long before Kubernetes kills the pod at 30 seconds.
	TimeoutShutdownDefault = 6 * time.Second

	// PostgresMaxOpenConnsDefault caps the connections one instance opens. Multiplied by the replica
	// count, the cap stays well under PostgreSQL's default max_connections of 100. That leaves room
	// for the migration job and a psql session. A spike waits inside the process for a free connection.
	PostgresMaxOpenConnsDefault = 20
	// PostgresMaxIdleConnsDefault matches the open limit, so a burst keeps the connections it is about
	// to reuse.
	PostgresMaxIdleConnsDefault = 20
	// PostgresPortDefault is PostgreSQL's standard port.
	PostgresPortDefault = 5432
	// PostgresTLSEnabledDefault encrypts the connection unless a deployment turns it off.
	PostgresTLSEnabledDefault = true

	// ServiceJsonKeysHostDefault reaches a JSON Keys service on the same host.
	ServiceJsonKeysHostDefault = "localhost"
	// ServiceJsonKeysPortDefault is the JSON Keys service's default gRPC port.
	ServiceJsonKeysPortDefault = 8080
	// ServiceGenaiPortDefault is the genai service's default gRPC port.
	ServiceGenaiPortDefault = 8080

	// RestTimeoutDefault bounds a request and the write of its response. A generation submit plans its
	// calls, then waits on genai for two of them at most, at 35 seconds each.
	RestTimeoutDefault = 90 * time.Second
	// RestMaxStreamDurationDefault bounds a streamed response to the 5 minutes OpenAI streams a call
	// for. A client resumes a longer stream from the last event it received.
	RestMaxStreamDurationDefault = 5 * time.Minute
	// RestMaxStreamsPerCallerDefault fits genai's 2 calls per user, the overlap of a reconnect, a project
	// load and a few tabs.
	RestMaxStreamsPerCallerDefault = 8

	// RestMaxRequestSizeDefault lets a batch through: up to 100 items, among them cells whose content
	// the core caps at 2 MiB each. The router applies it to every route.
	RestMaxRequestSizeDefault = 50 << 20
	// RestMaxItemRequestSizeDefault caps a route that writes one item. It fits a whole cell, with room
	// for the rest of the body.
	RestMaxItemRequestSizeDefault = 4 << 20
)

var (
	// PostgresDSN is the legacy URL used when PostgresHost is empty, formatted as
	//	postgres://<user>:<password>@<host>:<port>/<database>
	//
	// Configure the discrete PostgreSQL fields for new deployments.
	PostgresDSN = getEnv("POSTGRES_DSN")
	// PostgresHost is the PostgreSQL server hostname or IP address.
	PostgresHost = getEnv("POSTGRES_HOST")
	// PostgresPort is the PostgreSQL server port.
	PostgresPort = config.LoadEnv(getEnv("POSTGRES_PORT"), PostgresPortDefault, config.IntParser)
	// PostgresUser is the PostgreSQL login role.
	PostgresUser = getEnv("POSTGRES_USER")
	// PostgresPassword is the PostgreSQL login credential.
	PostgresPassword = getEnv("POSTGRES_PASSWORD")
	// PostgresDatabase is the PostgreSQL database name.
	PostgresDatabase = getEnv("POSTGRES_DATABASE")
	// PostgresTLSEnabled controls transport encryption for the PostgreSQL connection.
	PostgresTLSEnabled = config.LoadEnv(getEnv("POSTGRES_TLS_ENABLED"), PostgresTLSEnabledDefault, config.BoolParser)

	// PostgresMaxOpenConns is the maximum number of open connections to the database.
	PostgresMaxOpenConns = config.LoadEnv(getEnv("POSTGRES_MAX_OPEN_CONNS"), PostgresMaxOpenConnsDefault, config.IntParser)
	// PostgresMaxIdleConns is the maximum number of connections kept open while idle.
	PostgresMaxIdleConns = config.LoadEnv(getEnv("POSTGRES_MAX_IDLE_CONNS"), PostgresMaxIdleConnsDefault, config.IntParser)

	// ServiceJsonKeysHost is the hostname of the JSON Keys service, without scheme or port.
	ServiceJsonKeysHost = config.LoadEnv(getEnv("SERVICE_JSON_KEYS_HOST"), ServiceJsonKeysHostDefault, config.StringParser)
	// ServiceJsonKeysPort is the gRPC port of the JSON Keys service.
	ServiceJsonKeysPort = config.LoadEnv(getEnv("SERVICE_JSON_KEYS_PORT"), ServiceJsonKeysPortDefault, config.IntParser)
	// ServiceJsonKeysTokenPath is the file the kubelet projects this service's token for JSON Keys into.
	// When set, the JSON Keys connection sends that token with every call.
	ServiceJsonKeysTokenPath = getEnv("SERVICE_JSON_KEYS_TOKEN_PATH")

	// ServiceGenaiHost is the hostname of the genai service, without scheme or port. Empty leaves genai
	// unconfigured, and the generation routes answer 503.
	ServiceGenaiHost = getEnv("SERVICE_GENAI_HOST")
	// ServiceGenaiPort is the gRPC port of the genai service.
	ServiceGenaiPort = config.LoadEnv(getEnv("SERVICE_GENAI_PORT"), ServiceGenaiPortDefault, config.IntParser)
	// ServiceGenaiTokenPath is the file the kubelet projects this service's token for genai into. When set,
	// the genai connection sends that token with every call.
	ServiceGenaiTokenPath = getEnv("SERVICE_GENAI_TOKEN_PATH")

	// AppName is the name of the application, as it appears in logs and tracing.
	AppName = config.LoadEnv(getEnv("APP_NAME"), AppNameDefault, config.StringParser)
	// Otel enables OpenTelemetry tracing.
	Otel = config.LoadEnv(getEnv("OTEL"), false, config.BoolParser)

	// Rest configures the REST server from the REST_* variables.
	Rest = httpf.LoadServerConfig(getEnv, restDefaults())
	// RestMaxItemRequestSize caps the body of a route that writes one item, in bytes.
	RestMaxItemRequestSize = config.LoadEnv(
		getEnv("REST_MAX_ITEM_REQUEST_SIZE"), RestMaxItemRequestSizeDefault, config.Int64Parser,
	)
	// RestMaxStreamDuration bounds a streamed response.
	RestMaxStreamDuration = config.LoadEnv(
		getEnv("REST_MAX_STREAM_DURATION"), RestMaxStreamDurationDefault, config.DurationParser,
	)
	// RestMaxStreamsPerCaller caps the streams one caller holds open on an instance.
	RestMaxStreamsPerCaller = config.LoadEnv(
		getEnv("REST_MAX_STREAMS_PER_CALLER"), RestMaxStreamsPerCallerDefault, config.IntParser,
	)

	// ServiceTokenIssuer is the issuer of the service tokens the REST server accepts, as a token's "iss"
	// claim names it. When empty, the server takes every request without a service token.
	ServiceTokenIssuer = getEnv("SERVICE_TOKEN_ISSUER")
	// ServiceTokenAudience is the audience the REST server answers to. Each caller requests its token
	// for it.
	ServiceTokenAudience = getEnv("SERVICE_TOKEN_AUDIENCE")
	// ServiceTokenCallers are the service accounts allowed to call, as a comma-separated list of
	// system:serviceaccount:<namespace>:<name>.
	ServiceTokenCallers = config.LoadEnv(getEnv("SERVICE_TOKEN_CALLERS"), nil, config.SliceParser(config.StringParser))

	// GcloudProjectID is the Google Cloud project the service runs in. When set, the JSON Keys and
	// genai connections without a token path authenticate with Google ID tokens.
	GcloudProjectID = getEnv("GCLOUD_PROJECT_ID")

	// OTLPEndpoint is the standard OTEL_EXPORTER_OTLP_ENDPOINT. When set, traces and logs export
	// over OTLP/HTTP, which the OTEL_EXPORTER_OTLP_* variables configure. When empty, they go to
	// standard output. The OpenTelemetry SDK reads these variables without the prefix, so this one
	// is read without it too.
	OTLPEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// DowntimeStart is when a planned downtime of this service starts, in RFC 3339. From then
	// until it is removed, the service refuses work. Nil when none is planned.
	DowntimeStart = config.LoadEnv(getEnv("DOWNTIME_START"), nil, downtime.ParseStart)
)

// restDefaults keeps golib's REST defaults but for the versioned ping path and the request size.
func restDefaults() httpf.ServerConfig {
	defaults := httpf.ServerConfigDefault
	defaults.Liveness = "/v1/ping"
	defaults.Timeouts.Shutdown = TimeoutShutdownDefault
	defaults.Timeouts.Write, defaults.Timeouts.Request = RestTimeoutDefault, RestTimeoutDefault
	defaults.MaxRequestSize = RestMaxRequestSizeDefault

	return defaults
}
