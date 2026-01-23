package config

import (
	"os"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/logging"
	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/otel"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"

	"github.com/a-novel/service-template/internal/config/env"
)

const (
	OtelFlushTimeout = 2 * time.Second
)

// LoggerProd sends production-ready logs to Google Cloud environment.
var LoggerProd = loggingpresets.GRPCGcloud{
	Component: env.GcloudProjectId,
}

// LoggerDev prints logs in the console, pretty formatted.
var LoggerDev = loggingpresets.GRPCLocal{}

// LoggerDevHttp prints HTTP-level logs in the console, pretty formatted.
var LoggerDevHttp = &loggingpresets.LogLocal{
	Out: os.Stdout,
}

// LoggerProdHttp sends HTTP-level production-ready logs to Google Cloud environment.
var LoggerProdHttp = &loggingpresets.LogGcloud{
	ProjectId: env.GcloudProjectId,
}

var AppPresetDefault = App{
	App: Main{
		Name: env.AppName,
	},
	Grpc: Grpc{
		Port: env.GrpcPort,
		Ping: env.GrpcPing,
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
