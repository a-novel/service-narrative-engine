package config

import (
	"github.com/a-novel/service-narrative-engine/internal/config/env"
)

// AppPresetDefault is the configuration the service starts with. It reads every value from the
// environment. Once OTEL and OTEL_EXPORTER_OTLP_ENDPOINT are both set, it logs through the
// OpenTelemetry backend, and spans record each request and call, so the server writes no request log.
var AppPresetDefault = App{
	App: Main{
		Name:          env.AppName,
		DowntimeStart: env.DowntimeStart,
	},
	Dependencies: Dependencies{
		ServiceJSONKeys: env.ServiceJSONKeys,
		ServiceGenai:    env.ServiceGenai,
	},
	Permissions:  PermissionsConfigDefault,
	Caps:         CapsConfigDefault,
	Rest:         env.Rest,
	ServiceToken: env.ServiceToken,
	Limits: Limits{
		MaxItemRequestSize:        env.RestMaxItemRequestSize,
		MaxStreamDuration:         env.RestMaxStreamDuration,
		MaxStreamsPerCaller:       env.RestMaxStreamsPerCaller,
		RateLimitGeneration:       env.RestRateLimitGeneration,
		RateLimitGenerationRead:   env.RestRateLimitGenerationRead,
		MaxConcurrentProjectReads: env.MaxConcurrentProjectReads,
	},
	Generations: GenerationsConfigDefault,

	Otel:       env.Telemetry.Otel,
	Logger:     env.Telemetry.Logger,
	RestLogger: env.Telemetry.HTTPLogger,
	Postgres:   PostgresPresetDefault,
}
