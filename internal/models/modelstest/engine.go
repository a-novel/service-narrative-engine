// Package modelstest exposes shared model fixtures to tests across layers.
package modelstest

import _ "embed"

// WalkingSkeletonEngineDefinition is the canonical Engine definition used by tests.
//
//go:embed testdata/walking-skeleton.engine.json
var WalkingSkeletonEngineDefinition []byte
