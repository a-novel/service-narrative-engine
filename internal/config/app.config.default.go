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

// OtelFlushTimeout bounds how long the OpenTelemetry exporter waits to flush pending spans on
// shutdown.
const OtelFlushTimeout = 2 * time.Second

// OtelPresetDefault is the OpenTelemetry exporter of the servers. It exports nothing unless OTEL is
// set, then exports to Google Cloud when GCLOUD_PROJECT_ID is set, and to standard output otherwise.
var OtelPresetDefault = newOtelPreset()

// LoggerProdGrpc sends gRPC request logs to Google Cloud Logging.
var LoggerProdGrpc = loggingpresets.GRPCGcloud{
	Component: env.AppName,
	ProjectID: env.GcloudProjectID,
}

// LoggerDevGrpc prints gRPC request logs to the console for local development.
var LoggerDevGrpc = loggingpresets.GRPCLocal{}

// LoggerDev prints application and REST request logs to the console for local development.
var LoggerDev = &loggingpresets.LogLocal{
	Out: os.Stdout,
}

// LoggerProd sends application and REST request logs to Google Cloud Logging.
var LoggerProd = &loggingpresets.LogGcloud{
	ProjectID: env.GcloudProjectID,
}

// AppPresetDefault is the configuration the service starts with. It reads every value from the
// environment, and picks the Google Cloud logging and tracing backends once a project ID is set.
var AppPresetDefault = App{
	App: Main{
		Name:          env.AppName,
		DowntimeStart: env.DowntimeStart,
	},
	Grpc: env.Grpc,
	Rest: env.Rest,

	Otel:       OtelPresetDefault,
	Logger:     lo.Ternary[logging.Log](env.GcloudProjectID == "", LoggerDev, LoggerProd),
	GrpcLogger: lo.Ternary[logging.RPCConfig](env.GcloudProjectID == "", &LoggerDevGrpc, &LoggerProdGrpc),
	RestLogger: lo.Ternary[logging.HTTPConfig](
		env.GcloudProjectID == "",
		&loggingpresets.HTTPLocal{BaseLogger: LoggerDev},
		&loggingpresets.HTTPGcloud{BaseLogger: LoggerProd},
	),
	Postgres: PostgresPresetDefault,
}

// newOtelPreset picks the exporter [OtelPresetDefault] describes.
func newOtelPreset() otel.Config {
	switch {
	case !env.Otel:
		return &otelpresets.Disabled{}
	case env.GcloudProjectID == "":
		return &otelpresets.Local{FlushTimeout: OtelFlushTimeout}
	default:
		return &otelpresets.Gcloud{ProjectID: env.GcloudProjectID, FlushTimeout: OtelFlushTimeout}
	}
}
