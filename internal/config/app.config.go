package config

import (
	"time"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

// Dependencies configures the backing services this service calls.
type Dependencies struct {
	ServiceGenAIHost           string                    `json:"serviceGenAiHost"    yaml:"serviceGenAiHost"`
	ServiceGenAIPort           int                       `json:"serviceGenAiPort"    yaml:"serviceGenAiPort"`
	ServiceGenAICredentials    grpcf.CredentialsProvider `json:"-"                   yaml:"-"`
	ServiceJsonKeysHost        string                    `json:"serviceJSONKeysHost" yaml:"serviceJSONKeysHost"`
	ServiceJsonKeysPort        int                       `json:"serviceJSONKeysPort" yaml:"serviceJSONKeysPort"`
	ServiceJsonKeysCredentials grpcf.CredentialsProvider `json:"-"                   yaml:"-"`
}

// RestCors holds CORS configuration for the REST server.
type RestCors struct {
	AllowedOrigins   []string `json:"allowedOrigins"   yaml:"allowedOrigins"`
	AllowedHeaders   []string `json:"allowedHeaders"   yaml:"allowedHeaders"`
	AllowCredentials bool     `json:"allowCredentials" yaml:"allowCredentials"`
	MaxAge           int      `json:"maxAge"           yaml:"maxAge"`
}

// Main holds the top-level application settings.
type Main struct {
	// Name of the application, as it will appear in logs and tracing.
	Name string `json:"name" yaml:"name"`
}

// RestTimeouts holds timeout configuration for the REST server.
type RestTimeouts struct {
	Read       time.Duration `json:"read"       yaml:"read"`
	ReadHeader time.Duration `json:"readHeader" yaml:"readHeader"`
	Write      time.Duration `json:"write"      yaml:"write"`
	Idle       time.Duration `json:"idle"       yaml:"idle"`
	Request    time.Duration `json:"request"    yaml:"request"`
	// Shutdown bounds graceful request drain before remaining connections are closed.
	Shutdown time.Duration `json:"shutdown" yaml:"shutdown"`
}

// Rest holds the REST server configuration.
type Rest struct {
	// Port on which the REST server will listen for incoming requests.
	Port int `json:"port" yaml:"port"`
	// Timeouts groups the REST server timeout settings.
	Timeouts RestTimeouts `json:"timeouts" yaml:"timeouts"`
	// MaxRequestSize is the maximum size of an incoming request body.
	MaxRequestSize int64 `json:"maxRequestSize" yaml:"maxRequestSize"`
	// Cors holds the CORS configuration.
	Cors RestCors `json:"cors" yaml:"cors"`
}

// App aggregates the service's dependencies, HTTP server, observability, and database settings.
type App struct {
	App  Main `json:"app"  yaml:"app"`
	Rest Rest `json:"rest" yaml:"rest"`

	Dependencies Dependencies                      `json:"dependencies" yaml:"dependencies"`
	Permissions  serviceauthentication.Permissions `json:"permissions"  yaml:"permissions"`

	Otel       otel.Config        `json:"otel"       yaml:"otel"`
	Logger     logging.Log        `json:"logger"     yaml:"logger"`
	HttpLogger logging.HTTPConfig `json:"httpLogger" yaml:"httpLogger"`
	Postgres   postgres.Config    `json:"postgres"   yaml:"postgres"`
}
