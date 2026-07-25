package config

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/config"
)

//go:embed permissions.config.yaml
var defaultPermissionsFile []byte

// PermissionsConfigDefault is the built-in role and permission map.
var PermissionsConfigDefault = config.MustUnmarshal[serviceauthentication.Permissions](
	yaml.Unmarshal,
	defaultPermissionsFile,
)
