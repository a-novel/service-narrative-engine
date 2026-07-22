package config

import (
	"time"

	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

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

// PostgresPool holds the connection-pool limits applied to the database handle.
//
// They live here rather than on [postgres.Config] because that interface only knows how to open a
// connection, and the pool it returns carries Go's defaults — an unlimited number of open
// connections. The limits belong to the service, which knows how many of them it intends to use.
type PostgresPool struct {
	// MaxOpenConns is the maximum number of open connections to the database.
	MaxOpenConns int `json:"maxOpenConns" yaml:"maxOpenConns"`
	// MaxIdleConns is the maximum number of connections kept open while idle.
	MaxIdleConns int `json:"maxIdleConns" yaml:"maxIdleConns"`
}

// App is the root service configuration, aggregating the HTTP server, observability, and database
// settings.
type App struct {
	App  Main `json:"app"  yaml:"app"`
	Rest Rest `json:"rest" yaml:"rest"`

	Otel         otel.Config        `json:"otel"         yaml:"otel"`
	Log          logging.Log        `json:"log"          yaml:"log"`
	Logger       logging.RPCConfig  `json:"logger"       yaml:"logger"`
	HttpLogger   logging.HTTPConfig `json:"httpLogger"   yaml:"httpLogger"`
	Postgres     postgres.Config    `json:"postgres"     yaml:"postgres"`
	PostgresPool PostgresPool       `json:"postgresPool" yaml:"postgresPool"`
}
