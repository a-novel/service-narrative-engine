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
	"github.com/a-novel-kit/golib/servicetoken"

	"github.com/a-novel/service-narrative-engine/internal/config/env"
)

// OtelFlushTimeout bounds how long the OpenTelemetry exporter waits to flush pending spans on
// shutdown.
const OtelFlushTimeout = 2 * time.Second

// SentryFlushTimeout bounds how long a server waits on exit to send its pending Sentry events, inside
// the platform's shutdown grace period.
const SentryFlushTimeout = 2 * time.Second

// OtelPresetDefault is the OpenTelemetry exporter of the servers. It exports nothing unless OTEL is
// set, then exports over OTLP/HTTP when OTEL_EXPORTER_OTLP_ENDPOINT is set, and to standard output
// otherwise. The liveness probe produces no span.
var OtelPresetDefault = newOtelPreset()

// exportsOTLP reports whether the servers export over OTLP: OTEL is set, and so is the OTLP
// endpoint. Their logs then go through the same backend, and their spans record each request.
var exportsOTLP = env.Otel && env.OTLPEndpoint != ""

// LoggerDev prints application and REST request logs to the console for local development.
var LoggerDev = &loggingpresets.LogLocal{
	Out: os.Stdout,
}

// LoggerProd sends application logs through the OpenTelemetry backend, linked to their trace.
var LoggerProd = &loggingpresets.LogOTel{}

// AppPresetDefault is the configuration the service starts with. It reads every value from the
// environment, and logs through the OpenTelemetry backend when [exportsOTLP] holds. Spans then
// record each request and call, so the servers write no request log.
var AppPresetDefault = App{
	App: Main{
		Name:          env.AppName,
		DowntimeStart: env.DowntimeStart,
	},
	Dependencies: Dependencies{
		ServiceJsonKeysHost:        env.ServiceJsonKeysHost,
		ServiceJsonKeysPort:        env.ServiceJsonKeysPort,
		ServiceJsonKeysCredentials: newCredentials(env.ServiceJsonKeysTokenPath, env.ServiceJsonKeysHost),
		ServiceGenaiHost:           env.ServiceGenaiHost,
		ServiceGenaiPort:           env.ServiceGenaiPort,
		ServiceGenaiCredentials:    newCredentials(env.ServiceGenaiTokenPath, env.ServiceGenaiHost),
	},
	Permissions: PermissionsConfigDefault,
	Rest:        env.Rest,
	ServiceToken: servicetoken.Config{
		Issuer:   env.ServiceTokenIssuer,
		Audience: env.ServiceTokenAudience,
		Callers:  env.ServiceTokenCallers,
	},
	Limits: Limits{
		ItemMaxRequestSize: env.RestMaxItemRequestSize,
		MaxStreamDuration:  env.RestMaxStreamDuration,
	},
	Generations: GenerationsConfigDefault,

	Otel:   OtelPresetDefault,
	Sentry: env.Sentry,
	Logger: lo.Ternary[logging.Log](!exportsOTLP, LoggerDev, LoggerProd),
	RestLogger: lo.Ternary[logging.HTTPConfig](
		!exportsOTLP, &loggingpresets.HTTPLocal{BaseLogger: LoggerDev}, nil,
	),
	Postgres: PostgresPresetDefault,
}

// newCredentials authenticates the connection to the service at host with the service token at
// tokenPath. Each callee has its own audience, so each gets its own token. Without one, it sends a
// Google ID token when the service runs on Google Cloud, and nothing otherwise.
func newCredentials(tokenPath, host string) grpcf.CredentialsProvider {
	switch {
	case tokenPath != "":
		return &grpcf.ServiceTokenCredentialsProvider{ServiceTokenPath: tokenPath}
	case env.GcloudProjectID != "":
		return &grpcf.GcloudCredentialsProvider{Host: host}
	default:
		return &grpcf.LocalCredentialsProvider{}
	}
}

// newOtelPreset picks the exporter [OtelPresetDefault] describes.
func newOtelPreset() otel.Config {
	switch {
	case !env.Otel:
		return &otelpresets.Disabled{}
	case env.OTLPEndpoint == "":
		return &otelpresets.Local{FlushTimeout: OtelFlushTimeout}
	default:
		return &otelpresets.OTLP{UntracedPaths: []string{env.Rest.Liveness}, FlushTimeout: OtelFlushTimeout}
	}
}
