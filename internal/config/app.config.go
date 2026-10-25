package config

import (
	"time"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

// Main holds the top-level application settings.
type Main struct {
	// Name of the application, as it appears in logs and tracing.
	Name string `json:"name" yaml:"name"`
	// DowntimeStart is when a planned downtime starts; nil when none is planned. From then until
	// it is removed, the service refuses work and leaves its database alone.
	DowntimeStart *time.Time `json:"downtimeStart" yaml:"downtimeStart"`
}

// App is the complete configuration the service reads at startup.
type App struct {
	// App holds the top-level application settings.
	App Main `json:"app" yaml:"app"`
	// Grpc configures the gRPC server.
	Grpc grpcf.ServerConfig `json:"grpc" yaml:"grpc"`
	// Rest configures the REST server.
	Rest httpf.ServerConfig `json:"rest" yaml:"rest"`

	// Otel configures the OpenTelemetry exporter.
	Otel otel.Config `json:"otel" yaml:"otel"`
	// Logger is the application logger the REST handlers report their errors to.
	Logger logging.Log `json:"logger" yaml:"logger"`
	// GrpcLogger configures the gRPC request logging middleware.
	GrpcLogger logging.RPCConfig `json:"grpcLogger" yaml:"grpcLogger"`
	// RestLogger configures the REST request logging middleware.
	RestLogger logging.HTTPConfig `json:"restLogger" yaml:"restLogger"`
	// Postgres configures the PostgreSQL connection.
	Postgres postgres.Config `json:"postgres" yaml:"postgres"`
}
