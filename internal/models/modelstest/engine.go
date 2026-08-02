// Package modelstest exposes shared model fixtures to tests across layers.
package modelstest

import (
	_ "embed"
	"fmt"

	"github.com/goccy/go-yaml"
)

// WalkingSkeletonEngineDefinition is the canonical Engine definition used by tests.
//
//go:embed testdata/walking-skeleton.engine.yaml
var walkingSkeletonEngineDefinitionYAML []byte

var WalkingSkeletonEngineDefinition = MustJSONFromYAML(
	"walking-skeleton engine definition",
	walkingSkeletonEngineDefinitionYAML,
)

// MustJSONFromYAML converts static YAML test data to JSON and panics when the fixture is invalid.
func MustJSONFromYAML(name string, definition []byte) []byte {
	jsonDefinition, err := yaml.YAMLToJSON(definition)
	if err != nil {
		panic(fmt.Errorf("convert %s to json: %w", name, err))
	}

	return jsonDefinition
}
