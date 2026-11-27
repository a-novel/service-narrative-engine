package env

import (
	"cmp"
	"os"
	"strconv"

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

	// ServiceJSONKeysHostDefault reaches a JSON Keys service on the same host.
	ServiceJSONKeysHostDefault = "localhost"
	// ServiceJSONKeysPortDefault is the JSON Keys service's default gRPC port.
	ServiceJSONKeysPortDefault = 8080

	// RestMaxRequestSizeDefault lets a batch of items through: it is above the bytes a whole project stores.
	// The router applies it to every route.
	RestMaxRequestSizeDefault = 50 << 20
	// RestMaxItemRequestSizeDefault caps a route that writes one item. It fits the largest one, an instance's
	// content at [MaxContentSizeDefault], with room for the rest of the body.
	RestMaxItemRequestSizeDefault = 5 << 20

	// MaxContentSizeDefault fits a long novel several times over: one of 700 KB is under a fifth of it.
	MaxContentSizeDefault = 4 << 20
	// MaxDocumentDepthDefault is far above what a document nests by hand, and far under what a parser's stack
	// holds.
	MaxDocumentDepthDefault = 32
	// MaxInstancesPerProjectDefault leaves room for every plugin a project uses, several times each.
	MaxInstancesPerProjectDefault = 50
	// MaxLinksDefault gives a link to each cell of a long project.
	MaxLinksDefault = 20_000
	// MaxLinksSizeDefault fits [MaxLinksDefault] links of about a hundred bytes each.
	MaxLinksSizeDefault = 2 << 20
	// MaxLinkTargetLengthDefault fits an address a plugin builds from a few identifiers.
	MaxLinkTargetLengthDefault = 200
	// MaxLinkRelationshipSizeDefault fits a sentence or two in the writer's words.
	MaxLinkRelationshipSizeDefault = 1 << 10
)

var (
	// Postgres reaches PostgreSQL through the POSTGRES_* variables, with the fleet's pool bounds.
	Postgres = postgrespresets.LoadConnection(getEnv, postgrespresets.ConnectionDefault)

	// ServiceJSONKeys locates the JSON Keys gRPC server through the SERVICE_JSON_KEYS_* variables.
	ServiceJSONKeys = grpcf.LoadClientConfig(getEnv, "SERVICE_JSON_KEYS", grpcf.ClientConfig{
		Host: ServiceJSONKeysHostDefault,
		Port: ServiceJSONKeysPortDefault,
	})

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

	// MaxContentSize caps the content of an instance, in bytes of JSON.
	MaxContentSize = config.LoadEnv(getEnv("MAX_CONTENT_SIZE"), MaxContentSizeDefault, strconv.Atoi)
	// MaxDocumentDepth caps how deep a content nests its arrays and objects.
	MaxDocumentDepth = config.LoadEnv(getEnv("MAX_DOCUMENT_DEPTH"), MaxDocumentDepthDefault, strconv.Atoi)
	// MaxInstancesPerProject caps the instances a project holds.
	MaxInstancesPerProject = config.LoadEnv(
		getEnv("MAX_INSTANCES_PER_PROJECT"), MaxInstancesPerProjectDefault, strconv.Atoi,
	)
	// MaxLinks caps how many links a project holds.
	MaxLinks = config.LoadEnv(getEnv("MAX_LINKS"), MaxLinksDefault, strconv.Atoi)
	// MaxLinksSize caps the links of a project, in bytes of JSON.
	MaxLinksSize = config.LoadEnv(getEnv("MAX_LINKS_SIZE"), MaxLinksSizeDefault, strconv.Atoi)
	// MaxLinkTargetLength caps the target of either end of a link, in characters.
	MaxLinkTargetLength = config.LoadEnv(getEnv("MAX_LINK_TARGET_LENGTH"), MaxLinkTargetLengthDefault, strconv.Atoi)
	// MaxLinkRelationshipSize caps the relationship of a link, in bytes.
	MaxLinkRelationshipSize = config.LoadEnv(
		getEnv("MAX_LINK_RELATIONSHIP_SIZE"), MaxLinkRelationshipSizeDefault, strconv.Atoi,
	)

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
	defaults.MaxRequestSize = RestMaxRequestSizeDefault

	return defaults
}
