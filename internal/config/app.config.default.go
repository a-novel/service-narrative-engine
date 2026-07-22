package config

import (
	"os"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/logging"
	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/otel"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"

	"github.com/a-novel/service-narrative-engine/internal/config/env"
)

const (
	// OtelFlushTimeout bounds how long shutdown waits for OpenTelemetry to flush buffered data.
	OtelFlushTimeout = 2 * time.Second
)

// LoggerProd sends production-ready logs to a Google Cloud environment.
var LoggerProd = loggingpresets.GRPCGcloud{
	Component: env.GcloudProjectId,
}

// LoggerDev pretty-prints logs to the console.
var LoggerDev = loggingpresets.GRPCLocal{}

// LoggerDevHttp pretty-prints HTTP-level logs to the console.
var LoggerDevHttp = &loggingpresets.LogLocal{
	Out: os.Stdout,
}

// LoggerProdHttp sends production-ready HTTP-level logs to a Google Cloud environment.
var LoggerProdHttp = &loggingpresets.LogGcloud{
	ProjectId: env.GcloudProjectId,
}

// AppPresetDefault is the default application configuration, assembled from environment variables.
// Logging and telemetry fall back to local presets when no Google Cloud project is configured.
var AppPresetDefault = App{
	App: Main{
		Name: env.AppName,
	},
	Rest: Rest{
		Port: env.RestPort,
		Timeouts: RestTimeouts{
			Read:       env.RestTimeoutRead,
			ReadHeader: env.RestTimeoutReadHeader,
			Write:      env.RestTimeoutWrite,
			Idle:       env.RestTimeoutIdle,
			Request:    env.RestTimeoutRequest,
		},
		MaxRequestSize: env.RestMaxRequestSize,
		Cors: RestCors{
			AllowedOrigins:   env.CorsAllowedOrigins,
			AllowedHeaders:   env.CorsAllowedHeaders,
			AllowCredentials: env.CorsAllowCredentials,
			MaxAge:           env.CorsMaxAge,
		},
	},

	Otel: lo.If[otel.Config](!env.Otel, &otelpresets.Disabled{}).
		ElseIf(env.GcloudProjectId == "", &otelpresets.Local{
			FlushTimeout: OtelFlushTimeout,
		}).
		Else(&otelpresets.Gcloud{
			ProjectID:    env.GcloudProjectId,
			FlushTimeout: OtelFlushTimeout,
		}),
	Log:    lo.Ternary[logging.Log](env.GcloudProjectId == "", LoggerDevHttp, LoggerProdHttp),
	Logger: lo.Ternary[logging.RPCConfig](env.GcloudProjectId == "", &LoggerDev, &LoggerProd),
	HttpLogger: lo.Ternary[logging.HTTPConfig](
		env.GcloudProjectId == "",
		&loggingpresets.HTTPLocal{BaseLogger: LoggerDevHttp},
		&loggingpresets.HTTPGcloud{BaseLogger: LoggerProdHttp},
	),
	Postgres: PostgresPresetDefault,
}
