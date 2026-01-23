package config

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	authpkg "github.com/a-novel/service-authentication/v2/pkg"

	"github.com/a-novel-kit/golib/config"
)

//go:embed permissions.config.yaml
var defaultPermissionsFile []byte

var PermissionsConfigDefault = config.MustUnmarshal[authpkg.Permissions](yaml.Unmarshal, defaultPermissionsFile)
