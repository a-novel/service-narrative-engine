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

	ApiPortDefault              = 8080
	ApiTimeoutReadDefault       = 15 * time.Second
	ApiTimeoutReadHeaderDefault = 3 * time.Second
	ApiTimeoutWriteDefault      = 30 * time.Second
	ApiTimeoutIdleDefault       = 60 * time.Second
	ApiTimeoutRequestDefault    = 60 * time.Second
	ApiMaxRequestSizeDefault    = 2 << 20 // 2 MiB
	CorsAllowCredentialsDefault = false
	CorsMaxAgeDefault           = 3600
)

// Default values for environment variables, if applicable.
var (
	CorsAllowedOriginsDefault = []string{"*"}
	CorsAllowedHeadersDefault = []string{"*"}
)

// Raw values for environment variables.
var (
	postgresDsn     = getEnv("POSTGRES_DSN")
	postgresDsnTest = getEnv("POSTGRES_DSN_TEST")

	serviceJsonKeysHost = getEnv("SERVICE_JSON_KEYS_HOST")
	serviceJsonKeysPort = getEnv("SERVICE_JSON_KEYS_PORT")

	appName = getEnv("APP_NAME")
	otel    = getEnv("OTEL")

	apiPort              = getEnv("API_PORT")
	apiMaxRequestSize    = getEnv("API_MAX_REQUEST_SIZE")
	apiTimeoutRead       = getEnv("API_TIMEOUT_READ")
	apiTimeoutReadHeader = getEnv("API_TIMEOUT_READ_HEADER")
	apiTimeoutWrite      = getEnv("API_TIMEOUT_WRITE")
	apiTimeoutIdle       = getEnv("API_TIMEOUT_IDLE")
	apiTimeoutRequest    = getEnv("API_TIMEOUT_REQUEST")
	corsAllowedOrigins   = getEnv("API_CORS_ALLOWED_ORIGINS")
	corsAllowedHeaders   = getEnv("API_CORS_ALLOWED_HEADERS")
	corsAllowCredentials = getEnv("API_CORS_ALLOW_CREDENTIALS")
	corsMaxAge           = getEnv("API_CORS_MAX_AGE")

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
	// PostgresDsnTest is the url used to connect to the postgres database test instance.
	// Typically formatted as:
	//	postgres://<user>:<password>@<host>:<port>/<database>
	PostgresDsnTest = postgresDsnTest

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

	// ApiPort is the port on which the rest api server will listen for incoming requests.
	ApiPort              = config.LoadEnv(apiPort, ApiPortDefault, config.IntParser)
	ApiMaxRequestSize    = config.LoadEnv(apiMaxRequestSize, ApiMaxRequestSizeDefault, config.Int64Parser)
	ApiTimeoutRead       = config.LoadEnv(apiTimeoutRead, ApiTimeoutReadDefault, config.DurationParser)
	ApiTimeoutReadHeader = config.LoadEnv(apiTimeoutReadHeader, ApiTimeoutReadHeaderDefault, config.DurationParser)
	ApiTimeoutWrite      = config.LoadEnv(apiTimeoutWrite, ApiTimeoutWriteDefault, config.DurationParser)
	ApiTimeoutIdle       = config.LoadEnv(apiTimeoutIdle, ApiTimeoutIdleDefault, config.DurationParser)
	ApiTimeoutRequest    = config.LoadEnv(apiTimeoutRequest, ApiTimeoutRequestDefault, config.DurationParser)
	CorsAllowedOrigins   = config.LoadEnv(
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
