package config

import "github.com/goccy/go-yaml"

// unmarshalYAML decodes an embedded YAML file. An unknown key fails the decode, so a misspelled limit or
// grant stops the service at startup instead of reading as its zero value.
func unmarshalYAML(data []byte, value any) error {
	return yaml.UnmarshalWithOptions(data, value, yaml.DisallowUnknownField())
}
