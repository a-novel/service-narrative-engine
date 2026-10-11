package env

import (
	"os"

	"github.com/a-novel-kit/golib/config"
	"github.com/a-novel-kit/golib/httpf"
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

	ServiceGenAIHostDefault = "localhost"
	ServiceGenAIPortDefault = 8080

	ServiceJsonKeysHostDefault = "localhost"
	ServiceJsonKeysPortDefault = 8080

	// PostgresMaxOpenConnsDefault bounds the pool well below a stock PostgreSQL max_connections of
	// 100, leaving room for the migration job, a psql session, and the other services sharing the
	// instance. Go's own default is unlimited, which turns a traffic spike into connection refusals
	// for everything pointed at that database rather than queueing inside this process.
	PostgresMaxOpenConnsDefault = 20
	// PostgresMaxIdleConnsDefault matches the open limit so a burst does not close connections it is
	// about to reopen. Idle connections are cheap here; the TCP and TLS handshake they save is not.
	PostgresMaxIdleConnsDefault = 20
)

// Raw values for environment variables.
var (
	postgresDsn          = getEnv("POSTGRES_DSN")
	postgresMaxOpenConns = getEnv("POSTGRES_MAX_OPEN_CONNS")
	postgresMaxIdleConns = getEnv("POSTGRES_MAX_IDLE_CONNS")

	appName = getEnv("APP_NAME")
	otel    = getEnv("OTEL")

	serviceGenAIHost = getEnv("SERVICE_GENAI_HOST")
	serviceGenAIPort = getEnv("SERVICE_GENAI_PORT")

	serviceJsonKeysHost = getEnv("SERVICE_JSON_KEYS_HOST")
	serviceJsonKeysPort = getEnv("SERVICE_JSON_KEYS_PORT")

	gcloudProjectId = getEnv("GCLOUD_PROJECT_ID")
)

var (
	// PostgresDsn is the URL used to connect to the PostgreSQL database instance.
	// Typically formatted as:
	//	postgres://<user>:<password>@<host>:<port>/<database>
	PostgresDsn = postgresDsn

	// PostgresMaxOpenConns is the maximum number of open connections to the database.
	PostgresMaxOpenConns = config.LoadEnv(postgresMaxOpenConns, PostgresMaxOpenConnsDefault, config.IntParser)
	// PostgresMaxIdleConns is the maximum number of connections kept open while idle.
	PostgresMaxIdleConns = config.LoadEnv(postgresMaxIdleConns, PostgresMaxIdleConnsDefault, config.IntParser)

	// AppName is the name of the application, as it will appear in logs and tracing.
	AppName = config.LoadEnv(appName, AppNameDefault, config.StringParser)
	// Otel enables OpenTelemetry instrumentation.
	//
	// See: https://opentelemetry.io/
	Otel = config.LoadEnv(otel, false, config.BoolParser)

	// ServiceGenAIHost is the hostname of the service-genai gRPC server.
	ServiceGenAIHost = config.LoadEnv(serviceGenAIHost, ServiceGenAIHostDefault, config.StringParser)
	// ServiceGenAIPort is the port of the service-genai gRPC server.
	ServiceGenAIPort = config.LoadEnv(serviceGenAIPort, ServiceGenAIPortDefault, config.IntParser)

	// ServiceJsonKeysHost is the hostname of the JSON-keys gRPC server.
	ServiceJsonKeysHost = config.LoadEnv(serviceJsonKeysHost, ServiceJsonKeysHostDefault, config.StringParser)
	// ServiceJsonKeysPort is the port of the JSON-keys gRPC server.
	ServiceJsonKeysPort = config.LoadEnv(serviceJsonKeysPort, ServiceJsonKeysPortDefault, config.IntParser)

	// Rest configures the REST server from the REST_* variables.
	Rest = httpf.LoadServerConfig(getEnv, httpf.ServerConfigDefault)

	// GcloudProjectId configures the server for Google Cloud environment.
	//
	// See: https://docs.cloud.google.com/resource-manager/docs/creating-managing-projects
	GcloudProjectId = gcloudProjectId
)
