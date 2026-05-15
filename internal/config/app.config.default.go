package config

import (
	"os"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/logging"
	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/otel"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"

	"github.com/a-novel/service-narrative-engine/internal/config/env"
)

const (
	OtelFlushTimeout = 2 * time.Second
)

var LoggerDev = &loggingpresets.LogLocal{
	Out: os.Stdout,
}

var LoggerProd = &loggingpresets.LogGcloud{
	ProjectId: env.GcloudProjectId,
}

var AppPresetDefault = App{
	App: Main{
		Name: env.AppName,
	},
	Rest: Rest{
		Port:           env.RestPort,
		MaxRequestSize: env.RestMaxRequestSize,
		Timeouts: APITimeouts{
			Read:       env.RestTimeoutRead,
			ReadHeader: env.RestTimeoutReadHeader,
			Write:      env.RestTimeoutWrite,
			Idle:       env.RestTimeoutIdle,
			Request:    env.RestTimeoutRequest,
			RequestAI:  env.RestTimeoutRequestAI,
		},
		Cors: Cors{
			AllowedOrigins:   env.CorsAllowedOrigins,
			AllowedHeaders:   env.CorsAllowedHeaders,
			AllowCredentials: env.CorsAllowCredentials,
			MaxAge:           env.CorsMaxAge,
		},
	},

	DependenciesConfig: Dependencies{
		ServiceJsonKeysPort: env.ServiceJsonKeysPort,
		ServiceJsonKeysHost: env.ServiceJsonKeysHost,
		ServiceJsonKeysCredentials: lo.Ternary[grpcf.CredentialsProvider](
			env.GcloudProjectId == "",
			&grpcf.LocalCredentialsProvider{},
			&grpcf.GcloudCredentialsProvider{
				Host: env.ServiceJsonKeysHost,
			},
		),
	},
	Permissions: PermissionsConfigDefault,

	Otel: lo.If[otel.Config](!env.Otel, &otelpresets.Disabled{}).
		ElseIf(env.GcloudProjectId == "", &otelpresets.Local{
			FlushTimeout: OtelFlushTimeout,
		}).
		Else(&otelpresets.Gcloud{
			ProjectID:    env.GcloudProjectId,
			FlushTimeout: OtelFlushTimeout,
		}),
	Logger: lo.Ternary[logging.Log](env.GcloudProjectId == "", LoggerDev, LoggerProd),
	HttpLogger: lo.Ternary[logging.HTTPConfig](
		env.GcloudProjectId == "",
		&loggingpresets.HTTPLocal{
			BaseLogger: LoggerDev,
		},
		&loggingpresets.HTTPGcloud{
			BaseLogger: LoggerProd,
		},
	),
	Postgres: PostgresPresetDefault,
}
