package config

import (
	"github.com/a-novel-kit/golib/grpcf"

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
		ServiceJSONKeysHost:        env.ServiceJSONKeysHost,
		ServiceJSONKeysPort:        env.ServiceJSONKeysPort,
		ServiceJSONKeysCredentials: newCredentials(env.ServiceJSONKeysTokenPath, env.ServiceJSONKeysTLS),
		ServiceGenaiHost:           env.ServiceGenaiHost,
		ServiceGenaiPort:           env.ServiceGenaiPort,
		ServiceGenaiCredentials:    newCredentials(env.ServiceGenaiTokenPath, env.ServiceGenaiTLS),
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

// newCredentials authenticates a connection with the service token at tokenPath, over TLS once overTLS
// is set. Each callee has its own audience, so each gets its own token. Without one, it sends nothing
// and dials plaintext.
func newCredentials(tokenPath string, overTLS bool) grpcf.CredentialsProvider {
	if tokenPath == "" {
		return &grpcf.LocalCredentialsProvider{}
	}

	return &grpcf.ServiceTokenCredentialsProvider{ServiceTokenPath: tokenPath, TLS: overTLS}
}
