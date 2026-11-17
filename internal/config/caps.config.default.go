package config

import (
	_ "embed"

	"github.com/a-novel-kit/golib/config"
)

//go:embed caps.config.yaml
var defaultCapsFile []byte

// CapsConfigDefault is the built-in map of the caps each role's users take.
var CapsConfigDefault = config.MustUnmarshal[Caps](unmarshalYAML, defaultCapsFile)
