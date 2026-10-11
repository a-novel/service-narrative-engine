package config

import (
	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

// Dependencies configures the backing services this service calls.
type Dependencies struct {
	ServiceGenAIHost           string                    `json:"serviceGenAiHost"    yaml:"serviceGenAiHost"`
	ServiceGenAIPort           int                       `json:"serviceGenAiPort"    yaml:"serviceGenAiPort"`
	ServiceGenAICredentials    grpcf.CredentialsProvider `json:"-"                   yaml:"-"`
	ServiceJsonKeysHost        string                    `json:"serviceJSONKeysHost" yaml:"serviceJSONKeysHost"`
	ServiceJsonKeysPort        int                       `json:"serviceJSONKeysPort" yaml:"serviceJSONKeysPort"`
	ServiceJsonKeysCredentials grpcf.CredentialsProvider `json:"-"                   yaml:"-"`
}

// Main holds the top-level application settings.
type Main struct {
	// Name of the application, as it will appear in logs and tracing.
	Name string `json:"name" yaml:"name"`
}

// App aggregates the service's dependencies, HTTP server, observability, and database settings.
type App struct {
	App  Main               `json:"app"  yaml:"app"`
	Rest httpf.ServerConfig `json:"rest" yaml:"rest"`

	Dependencies Dependencies                      `json:"dependencies" yaml:"dependencies"`
	Permissions  serviceauthentication.Permissions `json:"permissions"  yaml:"permissions"`

	Otel       otel.Config        `json:"otel"       yaml:"otel"`
	Logger     logging.Log        `json:"logger"     yaml:"logger"`
	HttpLogger logging.HTTPConfig `json:"httpLogger" yaml:"httpLogger"`
	Postgres   postgres.Config    `json:"postgres"   yaml:"postgres"`
}
