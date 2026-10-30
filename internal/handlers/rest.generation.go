package handlers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestGeneration is a generation as the submit endpoint returns it: each of its calls with the genai
// generation that runs it, or with the cap it waits on.
type RestGeneration struct {
	// Kind is the kind of the generation, such as suggest.
	Kind string `json:"kind"`
	// Calls lists the generation's model calls, in order.
	Calls []*RestGenerationCall `json:"calls"`
}

// RestGenerationCall is one model call of a generation.
type RestGenerationCall struct {
	// Generation is the genai generation that runs the call. It is null while the call waits.
	Generation *RestGenerationResult `json:"generation"`
	// Waiting names the cap genai refused the call for. It is empty once genai admitted the call.
	Waiting string `json:"waiting,omitempty"`
	// Context says what the call reads, with the handles that map its output back.
	Context *RestGenerationContext `json:"context"`
}

// RestGenerationResult is one genai generation, as a submit, a read or a cancel finds it.
type RestGenerationResult struct {
	ID uuid.UUID `json:"id"`
	// Status is where the generation stands: pending, running, succeeded, failed or cancelled.
	Status string `json:"status"`

	// Error says why a failed generation failed.
	Error string `json:"error,omitempty"`
	// Output is the model's output, which passed the output schema of its kind, once it succeeded.
	Output json.RawMessage `json:"output,omitempty"`
	// Model names the model that wrote the output, as the provider names it, once it succeeded.
	Model string `json:"model,omitempty"`
}

// RestGenerationContext is what one call reads: its targets with their handles, and the scenes and
// layers it shows.
type RestGenerationContext struct {
	// Targets lists what the call writes, with the handles of the blocks the model may name.
	Targets []*RestContextTarget `json:"targets"`
	// Scenes lists the scenes the call shows, in narrative order, with the level each enters at.
	Scenes []*RestContextScene `json:"scenes"`
	// Layers lists the layers the call shows, in order.
	Layers []*RestContextLayer `json:"layers"`

	// Estimate is the token estimate of the call's prompt.
	Estimate int `json:"estimate"`
}

// RestContextTarget is one target of a call, as the model names it.
type RestContextTarget struct {
	Handle string `json:"handle"`

	// SceneID names the target's scene. It is null for a beat.
	SceneID *uuid.UUID `json:"sceneID"`
	// LayerID names the target's cell. It is null for a scene field or a beat.
	LayerID *uuid.UUID `json:"layerID"`
	// BeatID names the beat a consolidation recaps. It is null for any other target.
	BeatID *uuid.UUID `json:"beatID"`
	// Field names the target's scene field. It is empty for a cell or a beat.
	Field string `json:"field,omitempty"`

	// Mode is how a suggest call writes the target: write or revise. It is empty for a recap.
	Mode string `json:"mode,omitempty"`
	// Range is the span of a manuscript target's blocks the call may change.
	Range *RestRange `json:"range,omitempty"`
	// Kept lists the positions of the blocks in the range that stay unchanged.
	Kept []int `json:"kept,omitempty"`
	// Blocks maps the handle of each block the target shows inside its range to the block's position.
	Blocks []*RestContextBlock `json:"blocks,omitempty"`

	// Override marks a target whose current content came from the request.
	Override bool `json:"override"`
	// UpdatedAt is when the target's stored content was written. It is null for an override, and for a
	// cell never written. A recap write sends it back as the version the recap read.
	UpdatedAt *time.Time `json:"updatedAt"`
	// Sources lists, for each scene of the beat a consolidation recaps in order, the manuscript cell it
	// reads. A beat recap write sends them back.
	Sources []*BeatRecapSource `json:"sources,omitempty"`
}

// RestContextBlock is a block a target shows the model under a handle.
type RestContextBlock struct {
	Handle string `json:"handle"`
	// Position is the block's position in the target's cell, from 0.
	Position int `json:"position"`
}

// RestContextScene is one scene a call shows.
type RestContextScene struct {
	Handle  string    `json:"handle"`
	SceneID uuid.UUID `json:"sceneID"`

	// Level is how much of the scene the call shows, from full to outline.
	Level string `json:"level"`
}

// RestContextLayer is one layer a call shows.
type RestContextLayer struct {
	Handle  string    `json:"handle"`
	LayerID uuid.UUID `json:"layerID"`
}

// RestRange is a span of positions, from start up to end excluded.
type RestRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

func loadGeneration(generation *core.Generation) *RestGeneration {
	return &RestGeneration{
		Kind: generation.Kind,
		Calls: lo.Map(generation.Calls, func(call *core.GenerationCall, _ int) *RestGenerationCall {
			return &RestGenerationCall{
				Generation: loadGenerationResult(call.Result),
				Waiting:    call.Waiting,
				Context:    loadGenerationContext(call.Context),
			}
		}),
	}
}

// loadGenerationResult returns nil for a call genai has not admitted.
func loadGenerationResult(result *core.GenerationResult) *RestGenerationResult {
	if result == nil {
		return nil
	}

	return &RestGenerationResult{
		ID: result.ID, Status: result.Status, Error: result.Error, Output: result.Output, Model: result.Model,
	}
}

// loadGenerationContext keeps the scenes the call shows, and leaves out those it omits.
func loadGenerationContext(call *core.ContextCall) *RestGenerationContext {
	scenes := lo.FilterMap(call.Scenes, func(scene *core.ContextScene, _ int) (*RestContextScene, bool) {
		return &RestContextScene{Handle: scene.Handle, SceneID: scene.SceneID, Level: scene.Level},
			scene.Level != core.ContextLevelOmitted
	})

	return &RestGenerationContext{
		Targets: lo.Map(call.Targets, func(target *core.ContextTarget, _ int) *RestContextTarget {
			return loadContextTarget(target)
		}),
		Scenes: scenes,
		Layers: lo.Map(call.Layers, func(layer *core.ContextLayer, _ int) *RestContextLayer {
			return &RestContextLayer{Handle: layer.Handle, LayerID: layer.LayerID}
		}),
		Estimate: call.Estimate,
	}
}

// optionalID returns nil for an empty ID.
func optionalID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}

	return &id
}

func loadContextTarget(target *core.ContextTarget) *RestContextTarget {
	loaded := &RestContextTarget{
		Handle: target.Handle, SceneID: optionalID(target.SceneID), LayerID: optionalID(target.LayerID),
		BeatID: optionalID(target.BeatID), Field: target.Field, Mode: target.Mode,
		Kept: target.Kept, Override: target.Override, Sources: loadBeatRecapSources(target.Sources),
		Blocks: lo.Map(target.Blocks, func(block *core.ContextBlock, _ int) *RestContextBlock {
			return &RestContextBlock{Handle: block.Handle, Position: block.Position}
		}),
	}

	if target.Range != nil {
		loaded.Range = &RestRange{Start: target.Range.Start, End: target.Range.End}
	}

	if !target.UpdatedAt.IsZero() {
		loaded.UpdatedAt = &target.UpdatedAt
	}

	return loaded
}
