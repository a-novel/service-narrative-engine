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

var WalkingSkeletonEngineDefinition = func() []byte {
	jsonDefinition, err := yaml.YAMLToJSON(walkingSkeletonEngineDefinitionYAML)
	if err != nil {
		panic(fmt.Errorf("convert walking-skeleton engine definition to json: %w", err))
	}

	return jsonDefinition
}()
