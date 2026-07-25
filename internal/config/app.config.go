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
	ServiceJobsHost            string                    `json:"serviceJobsHost"     yaml:"serviceJobsHost"`
	ServiceJobsPort            int                       `json:"serviceJobsPort"     yaml:"serviceJobsPort"`
	ServiceJobsCredentials     grpcf.CredentialsProvider `json:"-"                   yaml:"-"`
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

// HTTPClient sizes the connection pool of the one client every outbound provider call shares.
type HTTPClient struct {
	// MaxIdleConns is the number of idle connections kept across every provider host.
	MaxIdleConns int `json:"maxIdleConns" yaml:"maxIdleConns"`
	// MaxIdleConnsPerHost is the number of idle connections kept for one provider host.
	// Keep it at least as large as provider concurrency so calls reuse existing connections.
	MaxIdleConnsPerHost int `json:"maxIdleConnsPerHost" yaml:"maxIdleConnsPerHost"`
}

// Worker holds the process-level controls for claiming and running narrative jobs.
type Worker struct {
	// Concurrency is the number of claim pollers and jobs that may run at once.
	Concurrency int `json:"concurrency" yaml:"concurrency"`
	// PollInterval is the delay after an empty or failed claim.
	PollInterval time.Duration `json:"pollInterval" yaml:"pollInterval"`
	// JobDeadline bounds one handler invocation.
	JobDeadline time.Duration `json:"jobDeadline" yaml:"jobDeadline"`
	// Lease is the claim lifetime derived from JobDeadline plus settlement slack.
	Lease time.Duration `json:"lease" yaml:"lease"`
	// DrainBudget bounds in-flight execution after the claim loop stops.
	DrainBudget time.Duration `json:"drainBudget" yaml:"drainBudget"`
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
	HTTPClient HTTPClient         `json:"httpClient" yaml:"httpClient"`
	Worker     Worker             `json:"worker"     yaml:"worker"`
}
