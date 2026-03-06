package env

import (
	"os"
	"time"

	"github.com/a-novel-kit/golib/config"
)

// prefix allows setting a custom prefix to all configuration environment variables.
// This is useful when importing the package in another project, when env variable names
// might conflict with the source project.
var prefix = os.Getenv("SERVICE_NARRATIVE_ENGINE_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

// Default values for environment variables, if applicable.
const (
	AppNameDefault = "service-narrative-engine"

	ServiceJsonKeysHostDefault = "localhost"
	ServiceJsonKeysPortDefault = 8080

	RestPortDefault              = 8080
	RestTimeoutReadDefault       = 15 * time.Second
	RestTimeoutReadHeaderDefault = 3 * time.Second
	RestTimeoutWriteDefault      = 15 * time.Minute
	RestTimeoutIdleDefault       = 0
	RestTimeoutRequestDefault    = 60 * time.Second
	RestTimeoutRequestAIDefault  = 15 * time.Minute
	RestMaxRequestSizeDefault    = 5 << 20 // 5 MiB
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

	serviceJsonKeysHost = getEnv("SERVICE_JSON_KEYS_HOST")
	serviceJsonKeysPort = getEnv("SERVICE_JSON_KEYS_PORT")

	appName = getEnv("APP_NAME")
	otel    = getEnv("OTEL")

	restPort              = getEnv("REST_PORT")
	restMaxRequestSize    = getEnv("REST_MAX_REQUEST_SIZE")
	restTimeoutRead       = getEnv("REST_TIMEOUT_READ")
	restTimeoutReadHeader = getEnv("REST_TIMEOUT_READ_HEADER")
	restTimeoutWrite      = getEnv("REST_TIMEOUT_WRITE")
	restTimeoutIdle       = getEnv("REST_TIMEOUT_IDLE")
	restTimeoutRequest    = getEnv("REST_TIMEOUT_REQUEST")
	restTimeoutRequestAI  = getEnv("REST_TIMEOUT_REQUEST_AI")
	corsAllowedOrigins    = getEnv("REST_CORS_ALLOWED_ORIGINS")
	corsAllowedHeaders    = getEnv("REST_CORS_ALLOWED_HEADERS")
	corsAllowCredentials  = getEnv("REST_CORS_ALLOW_CREDENTIALS")
	corsMaxAge            = getEnv("REST_CORS_MAX_AGE")

	gcloudProjectId = getEnv("GCLOUD_PROJECT_ID")

	openAiToken   = getEnv("OPENAI_API_KEY")
	openAiBaseUrl = getEnv("OPENAI_BASE_URL")
	openAiModel   = getEnv("OPENAI_MODEL")

	devMode = getEnv("DEV_MODE")
	version = getEnv("VERSION")
)

var (
	// PostgresDsn is the url used to connect to the postgres database instance.
	// Typically formatted as:
	//	postgres://<user>:<password>@<host>:<port>/<database>
	PostgresDsn = postgresDsn

	// ServiceJsonKeysHost points to the host name (without protocol / port) on which the JSON Keys Service is hosted.
	//
	// See https://github.com/a-novel/service-json-keys
	ServiceJsonKeysHost = config.LoadEnv(serviceJsonKeysHost, ServiceJsonKeysHostDefault, config.StringParser)
	// ServiceJsonKeysPort points to the port on which the JSON Keys Service is hosted.
	//
	// See https://github.com/a-novel/service-json-keys
	ServiceJsonKeysPort = config.LoadEnv(serviceJsonKeysPort, ServiceJsonKeysPortDefault, config.IntParser)

	// AppName is the name of the application, as it will appear in logs and tracing.
	AppName = config.LoadEnv(appName, AppNameDefault, config.StringParser)
	// Otel flag configures whether to use Open Telemetry or not.
	//
	// See: https://opentelemetry.io/
	Otel = config.LoadEnv(otel, false, config.BoolParser)

	RestPort              = config.LoadEnv(restPort, RestPortDefault, config.IntParser)
	RestMaxRequestSize    = config.LoadEnv(restMaxRequestSize, RestMaxRequestSizeDefault, config.Int64Parser)
	RestTimeoutRead       = config.LoadEnv(restTimeoutRead, RestTimeoutReadDefault, config.DurationParser)
	RestTimeoutReadHeader = config.LoadEnv(restTimeoutReadHeader, RestTimeoutReadHeaderDefault, config.DurationParser)
	RestTimeoutWrite      = config.LoadEnv(restTimeoutWrite, RestTimeoutWriteDefault, config.DurationParser)
	RestTimeoutIdle       = config.LoadEnv(restTimeoutIdle, RestTimeoutIdleDefault, config.DurationParser)
	RestTimeoutRequest    = config.LoadEnv(restTimeoutRequest, RestTimeoutRequestDefault, config.DurationParser)
	RestTimeoutRequestAI  = config.LoadEnv(restTimeoutRequestAI, RestTimeoutRequestAIDefault, config.DurationParser)
	CorsAllowedOrigins    = config.LoadEnv(
		corsAllowedOrigins, CorsAllowedOriginsDefault, config.SliceParser(config.StringParser),
	)
	CorsAllowedHeaders = config.LoadEnv(
		corsAllowedHeaders, CorsAllowedHeadersDefault, config.SliceParser(config.StringParser),
	)
	CorsAllowCredentials = config.LoadEnv(corsAllowCredentials, CorsAllowCredentialsDefault, config.BoolParser)
	CorsMaxAge           = config.LoadEnv(corsMaxAge, CorsMaxAgeDefault, config.IntParser)

	// GcloudProjectId configures the server for Google Cloud environment.
	//
	// See: https://docs.cloud.google.com/resource-manager/docs/creating-managing-projects
	GcloudProjectId = gcloudProjectId

	OpenAiBaseUrl = openAiBaseUrl
	OpenAiModel   = openAiModel
	OpenAiApiKey  = openAiToken

	// DevMode enables development mode features, such as preversioning for system modules.
	DevMode = config.LoadEnv(devMode, false, config.BoolParser)
	// Version is the current version of the service. This is required for system module loading.
	// It has no default value and MUST be provided.
	Version = version
)
