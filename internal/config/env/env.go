package env

import (
	"os"
	"time"

	"github.com/a-novel-kit/golib/config"
)

// prefix is prepended to every configuration environment variable name. Setting
// SERVICE_NARRATIVE_ENGINE_ENV_PREFIX avoids name clashes when another project embeds this service.
var prefix = os.Getenv("SERVICE_NARRATIVE_ENGINE_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

// Default values for environment variables, if applicable.
const (
	AppNameDefault = "service-narrative-engine"

	RestPortDefault              = 8080
	RestTimeoutReadDefault       = 15 * time.Second
	RestTimeoutReadHeaderDefault = 3 * time.Second
	RestTimeoutWriteDefault      = 30 * time.Second
	RestTimeoutIdleDefault       = 60 * time.Second
	RestTimeoutRequestDefault    = 60 * time.Second
	RestMaxRequestSizeDefault    = 2 << 20 // 2 MiB
	CorsAllowCredentialsDefault  = false
	CorsMaxAgeDefault            = 3600
)

// Default values for environment variables, if applicable.
var (
	CorsAllowedOriginsDefault = []string{"*"}
	CorsAllowedHeadersDefault = []string{"*"}
)

// Raw values for environment variables.
var (
	postgresDsn = getEnv("POSTGRES_DSN")

	appName = getEnv("APP_NAME")
	otel    = getEnv("OTEL")

	restPort              = getEnv("REST_PORT")
	restTimeoutRead       = getEnv("REST_TIMEOUT_READ")
	restTimeoutReadHeader = getEnv("REST_TIMEOUT_READ_HEADER")
	restTimeoutWrite      = getEnv("REST_TIMEOUT_WRITE")
	restTimeoutIdle       = getEnv("REST_TIMEOUT_IDLE")
	restTimeoutRequest    = getEnv("REST_TIMEOUT_REQUEST")
	restMaxRequestSize    = getEnv("REST_MAX_REQUEST_SIZE")

	corsAllowedOrigins   = getEnv("REST_CORS_ALLOWED_ORIGINS")
	corsAllowedHeaders   = getEnv("REST_CORS_ALLOWED_HEADERS")
	corsAllowCredentials = getEnv("REST_CORS_ALLOW_CREDENTIALS")
	corsMaxAge           = getEnv("REST_CORS_MAX_AGE")

	gcloudProjectId = getEnv("GCLOUD_PROJECT_ID")
)

var (
	// PostgresDsn is the URL used to connect to the PostgreSQL database instance.
	// Typically formatted as:
	//	postgres://<user>:<password>@<host>:<port>/<database>
	PostgresDsn = postgresDsn

	// AppName is the name of the application, as it will appear in logs and tracing.
	AppName = config.LoadEnv(appName, AppNameDefault, config.StringParser)
	// Otel enables OpenTelemetry instrumentation.
	//
	// See: https://opentelemetry.io/
	Otel = config.LoadEnv(otel, false, config.BoolParser)

	// RestPort is the port on which the REST server will listen for incoming requests.
	RestPort = config.LoadEnv(restPort, RestPortDefault, config.IntParser)
	// RestTimeoutRead is the maximum duration for reading an incoming REST request.
	RestTimeoutRead = config.LoadEnv(restTimeoutRead, RestTimeoutReadDefault, config.DurationParser)
	// RestTimeoutReadHeader is the maximum duration for reading the headers of an incoming REST request.
	RestTimeoutReadHeader = config.LoadEnv(restTimeoutReadHeader, RestTimeoutReadHeaderDefault, config.DurationParser)
	// RestTimeoutWrite is the maximum duration for writing a REST response.
	RestTimeoutWrite = config.LoadEnv(restTimeoutWrite, RestTimeoutWriteDefault, config.DurationParser)
	// RestTimeoutIdle is the maximum duration to wait for the next request when keep-alives are enabled.
	RestTimeoutIdle = config.LoadEnv(restTimeoutIdle, RestTimeoutIdleDefault, config.DurationParser)
	// RestTimeoutRequest is the maximum duration for processing an incoming REST request.
	RestTimeoutRequest = config.LoadEnv(restTimeoutRequest, RestTimeoutRequestDefault, config.DurationParser)
	// RestMaxRequestSize is the maximum size of an incoming REST request body.
	RestMaxRequestSize = config.LoadEnv(restMaxRequestSize, RestMaxRequestSizeDefault, config.Int64Parser)

	// CorsAllowedOrigins lists the origins allowed to access the REST API.
	CorsAllowedOrigins = config.LoadEnv(
		corsAllowedOrigins, CorsAllowedOriginsDefault, config.SliceParser(config.StringParser),
	)
	// CorsAllowedHeaders lists the headers allowed in CORS requests.
	CorsAllowedHeaders = config.LoadEnv(
		corsAllowedHeaders, CorsAllowedHeadersDefault, config.SliceParser(config.StringParser),
	)
	// CorsAllowCredentials configures whether CORS requests can include credentials.
	CorsAllowCredentials = config.LoadEnv(corsAllowCredentials, CorsAllowCredentialsDefault, config.BoolParser)
	// CorsMaxAge sets the maximum age (in seconds) for CORS preflight cache.
	CorsMaxAge = config.LoadEnv(corsMaxAge, CorsMaxAgeDefault, config.IntParser)

	// GcloudProjectId configures the server for Google Cloud environment.
	//
	// See: https://docs.cloud.google.com/resource-manager/docs/creating-managing-projects
	GcloudProjectId = gcloudProjectId
)
