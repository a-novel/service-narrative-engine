package env

import (
	"cmp"
	"os"
	"strconv"
	"time"

	"github.com/a-novel-kit/golib/config"
	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"
	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"
	"github.com/a-novel-kit/golib/servicetoken"
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

	// ServiceJSONKeysHostDefault reaches a JSON Keys service on the same host.
	ServiceJSONKeysHostDefault = "localhost"
	// ServiceJSONKeysPortDefault is the JSON Keys service's default gRPC port.
	ServiceJSONKeysPortDefault = 8080
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
	// RestRateLimitGenerationDefault caps a user's generation submits. Each one reads the project and plans
	// its calls before genai's per-user quota applies, so a burst costs this service first.
	RestRateLimitGenerationDefault = config.Must(httpf.ParseRate("30/1m"))
	// RestRateLimitGenerationReadDefault caps a user's reads of a call. Each unsettled read asks the provider
	// for the call's state through genai, and a client polling twice a second sees a finished call at once.
	RestRateLimitGenerationReadDefault = config.Must(httpf.ParseRate("2/1s"))
)

var (
	// Postgres reaches PostgreSQL through the POSTGRES_* variables, with the fleet's pool bounds.
	Postgres = postgrespresets.LoadConnection(getEnv, postgrespresets.ConnectionDefault)

	// ServiceJSONKeys locates the JSON Keys gRPC server through the SERVICE_JSON_KEYS_* variables.
	ServiceJSONKeys = grpcf.LoadClientConfig(getEnv, "SERVICE_JSON_KEYS", grpcf.ClientConfig{
		Host: ServiceJSONKeysHostDefault,
		Port: ServiceJSONKeysPortDefault,
	})
	// ServiceGenai locates the genai gRPC server through the SERVICE_GENAI_* variables. Without a host,
	// genai stays unconfigured, and the generation routes answer 503.
	ServiceGenai = grpcf.LoadClientConfig(getEnv, "SERVICE_GENAI", grpcf.ClientConfig{Port: ServiceGenaiPortDefault})

	// AppName is the name of the application, as it appears in logs and tracing.
	AppName = cmp.Or(getEnv("APP_NAME"), AppNameDefault)
	// Telemetry is the OpenTelemetry backend that OTEL and OTEL_EXPORTER_OTLP_ENDPOINT select, with the
	// loggers that match it.
	Telemetry = otelpresets.LoadTelemetry(getEnv, otelpresets.ServerFlushTimeout)

	// Rest configures the REST server from the REST_* variables.
	Rest = httpf.LoadServerConfig(getEnv, restDefaults())
	// RestMaxItemRequestSize caps the body of a route that writes one item, in bytes.
	RestMaxItemRequestSize = config.LoadEnv(
		getEnv("REST_MAX_ITEM_REQUEST_SIZE"), RestMaxItemRequestSizeDefault, config.Int64Parser,
	)
	// RestMaxStreamDuration bounds a streamed response.
	RestMaxStreamDuration = config.LoadEnv(
		getEnv("REST_MAX_STREAM_DURATION"), RestMaxStreamDurationDefault, time.ParseDuration,
	)
	// RestMaxStreamsPerCaller caps the streams one caller holds open on an instance.
	RestMaxStreamsPerCaller = config.LoadEnv(
		getEnv("REST_MAX_STREAMS_PER_CALLER"), RestMaxStreamsPerCallerDefault, strconv.Atoi,
	)
	// RestRateLimitGeneration caps each signed-in user's generation submits.
	RestRateLimitGeneration = config.LoadEnv(
		getEnv("REST_RATE_LIMIT_GENERATION"), RestRateLimitGenerationDefault, httpf.ParseRate,
	)
	// RestRateLimitGenerationRead caps each signed-in user's reads of a call, on each read route.
	RestRateLimitGenerationRead = config.LoadEnv(
		getEnv("REST_RATE_LIMIT_GENERATION_READ"), RestRateLimitGenerationReadDefault, httpf.ParseRate,
	)
	// MaxConcurrentProjectReads caps the reads of a whole project an instance runs at once. Unset, 0 lets
	// core.NewProjectReadGate size them to the instance's memory.
	MaxConcurrentProjectReads = config.LoadEnv(getEnv("MAX_CONCURRENT_PROJECT_READS"), 0, strconv.Atoi)

	// ServiceToken names the service tokens the REST server accepts. Without an issuer, the server
	// refuses to start unless SERVICE_ISSUER_OPTIONAL is set.
	ServiceToken = servicetoken.LoadConfig(getEnv)

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
