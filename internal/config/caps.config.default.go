package config

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	"github.com/a-novel-kit/golib/config"
)

//go:embed caps.config.yaml
var defaultCapsFile []byte

// CapsConfigDefault is the built-in map of the caps each role's users take. An unknown key fails the
// decode, so a misspelled limit cannot silently lift a cap.
var CapsConfigDefault = config.MustUnmarshal[Caps](func(data []byte, value any) error {
	return yaml.UnmarshalWithOptions(data, value, yaml.DisallowUnknownField())
}, defaultCapsFile)
