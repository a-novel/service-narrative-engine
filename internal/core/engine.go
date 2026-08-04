package core

import "github.com/a-novel/service-narrative-engine/internal/lib"

var (
	// ErrEngineDefinitionInvalid reports an Engine Version definition that cannot
	// safely drive generation or local validation.
	ErrEngineDefinitionInvalid = lib.ErrEngineDefinitionInvalid
	// ErrEngineStepNotFound reports a requested step absent from the Engine Version.
	ErrEngineStepNotFound = lib.ErrEngineStepNotFound
)
