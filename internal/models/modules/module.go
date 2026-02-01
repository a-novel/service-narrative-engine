package modules

import (
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/google/jsonschema-go/jsonschema"
)

type SystemModule struct {
	ID          string            `yaml:"id"`
	Namespace   string            `yaml:"namespace"`
	Description string            `yaml:"description"`
	Schema      jsonschema.Schema `yaml:"schema"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
//
// The data is stored in YAML for maintainability and readability. However, the jsonschema.Schema it encodes relies on
// its own unmarshalling mechanisms from JSON. To make sure everything gets translated correctly, we force a round-trip:
// YANL -> GO -> JSON -> GO. This way, the final value comes from JSON and uses the proper unmarshal mechanisms.
func (module *SystemModule) UnmarshalYAML(data []byte) error {
	var init any

	// Round-trip from YAML to JSON using Go.
	err := yaml.Unmarshal(data, &init)
	if err != nil {
		return fmt.Errorf("unmarshal yaml: %w", err)
	}

	// Now remarshal as JSON.
	initMarshalled, err := json.Marshal(init)
	if err != nil {
		return fmt.Errorf("marshal init: %w", err)
	}

	var final struct {
		ID          string            `json:"id"`
		Namespace   string            `json:"namespace"`
		Description string            `json:"description"`
		Schema      jsonschema.Schema `json:"schema"`
	}

	err = json.Unmarshal(initMarshalled, &final)
	if err != nil {
		return fmt.Errorf("unmarshal final: %w", err)
	}

	module.ID = final.ID
	module.Namespace = final.Namespace
	module.Description = final.Description
	module.Schema = final.Schema

	return nil
}

var _ yaml.BytesUnmarshaler = &SystemModule{}
