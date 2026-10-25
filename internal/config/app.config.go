package config

import (
	"time"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/sentryf"
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
	// Composer bounds the size of each project.
	Composer Composer `json:"composer" yaml:"composer"`
	// Rest configures the REST server.
	Rest httpf.ServerConfig `json:"rest" yaml:"rest"`

	// Otel configures the OpenTelemetry exporter.
	Otel otel.Config `json:"otel" yaml:"otel"`
	// Sentry configures the reporter that sends server faults to Sentry.
	Sentry sentryf.Config `json:"sentry" yaml:"sentry"`
	// Logger is the application logger the REST handlers report their errors to.
	Logger logging.Log `json:"logger" yaml:"logger"`
	// RestLogger configures the REST request logging middleware.
	RestLogger logging.HTTPConfig `json:"restLogger" yaml:"restLogger"`
	// Postgres configures the PostgreSQL connection.
	Postgres postgres.Config `json:"postgres" yaml:"postgres"`
}
