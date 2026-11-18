package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Generation is a generation as the submit endpoint returns it: each of its calls with the genai
// generation that runs it, or with the cap it waits on.
type Generation struct {
	// Kind is the kind of the generation, such as suggest.
	Kind string `json:"kind"`
	// Calls lists the generation's model calls, in order.
	Calls []*GenerationCall `json:"calls"`
}

// GenerationCall is one model call of a generation.
type GenerationCall struct {
	// Generation is the genai generation that runs the call. It is null while the call waits.
	Generation *GenerationResult `json:"generation"`
	// Waiting names the cap genai refused the call for. It is empty once genai admitted the call.
	Waiting string `json:"waiting,omitempty"`
	// Context says what the call reads, with the handles that map its output back.
	Context *GenerationContext `json:"context"`
}

// GenerationResult is one genai generation, as a submit, a read or a cancel finds it.
type GenerationResult struct {
	ID uuid.UUID `json:"id"`
	// Status is where the generation stands: pending, running, succeeded, failed or cancelled.
	Status string `json:"status"`

	// Error says why a failed generation failed.
	Error string `json:"error,omitempty"`
	// FailureClass says why a failed generation failed, so a client decides whether to resend it: rejected,
	// provider_failed, not_started, outcome_unknown or invalid_output. It is empty for a failure genai did not
	// classify.
	FailureClass string `json:"failureClass,omitempty"`
	// Output is the model's output, which passed the output schema of its kind, once it succeeded.
	Output json.RawMessage `json:"output,omitempty"`
	// Model names the model that wrote the output, as the provider names it, once it succeeded.
	Model string `json:"model,omitempty"`
}

// GenerationContext is what one call reads: its targets with their handles, and the scenes and
// layers it shows.
type GenerationContext struct {
	// Targets lists what the call writes, with the handles of the blocks the model may name.
	Targets []*ContextTarget `json:"targets"`
	// Scenes lists the scenes the call shows, in narrative order, with the level each enters at.
	Scenes []*ContextScene `json:"scenes"`
	// Layers lists the layers the call shows, in order.
	Layers []*ContextLayer `json:"layers"`

	// Estimate is the token estimate of the call's prompt.
	Estimate int `json:"estimate"`
}

// ContextTarget is one target of a call, as the model names it.
type ContextTarget struct {
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
	Range *Range `json:"range,omitempty"`
	// Kept lists the positions of the blocks in the range that stay unchanged.
	Kept []int `json:"kept,omitempty"`
	// Blocks maps the handle of each block the target shows inside its range to the block's position.
	Blocks []*ContextBlock `json:"blocks,omitempty"`

	// Override marks a target whose current content came from the request.
	Override bool `json:"override"`
	// UpdatedAt is when the target's stored content was written. It is null for an override, and for a
	// cell never written. A recap write sends it back as the version the recap read.
	UpdatedAt *time.Time `json:"updatedAt"`
	// Sources lists, for each scene of the beat a consolidation recaps in order, the manuscript cell it
	// reads. A beat recap write sends them back.
	Sources []*BeatRecapSource `json:"sources,omitempty"`
}

// ContextBlock is a block a target shows the model under a handle.
type ContextBlock struct {
	Handle string `json:"handle"`
	// Position is the block's position in the target's cell, from 0.
	Position int `json:"position"`
}

// ContextScene is one scene a call shows.
type ContextScene struct {
	Handle  string    `json:"handle"`
	SceneID uuid.UUID `json:"sceneID"`

	// Level is how much of the scene the call shows, from full to outline.
	Level string `json:"level"`
}

// ContextLayer is one layer a call shows.
type ContextLayer struct {
	Handle  string    `json:"handle"`
	LayerID uuid.UUID `json:"layerID"`
}

// Range is a span of positions, from start up to end excluded.
type Range struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

func loadGeneration(generation *core.Generation) *Generation {
	return &Generation{
		Kind: generation.Kind,
		Calls: lo.Map(generation.Calls, func(call *core.GenerationCall, _ int) *GenerationCall {
			return &GenerationCall{
				Generation: loadGenerationResult(call.Result),
				Waiting:    call.Waiting,
				Context:    loadGenerationContext(call.Context),
			}
		}),
	}
}

// loadGenerationResult returns nil for a call genai has not admitted.
func loadGenerationResult(result *core.GenerationResult) *GenerationResult {
	if result == nil {
		return nil
	}

	return &GenerationResult{
		ID: result.ID, Status: result.Status, Error: result.Error, FailureClass: result.FailureClass,
		Output: result.Output, Model: result.Model,
	}
}

// loadGenerationContext keeps the scenes the call shows, and leaves out those it omits.
func loadGenerationContext(call *core.ContextCall) *GenerationContext {
	scenes := make([]*ContextScene, 0, len(call.Scenes))

	for _, scene := range call.Scenes {
		if scene.Level != core.ContextLevelOmitted {
			scenes = append(scenes, &ContextScene{Handle: scene.Handle, SceneID: scene.SceneID, Level: scene.Level})
		}
	}

	return &GenerationContext{
		Targets: lo.Map(call.Targets, func(target *core.ContextTarget, _ int) *ContextTarget {
			return loadContextTarget(target)
		}),
		Scenes: scenes,
		Layers: lo.Map(call.Layers, func(layer *core.ContextLayer, _ int) *ContextLayer {
			return &ContextLayer{Handle: layer.Handle, LayerID: layer.LayerID}
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

func loadContextTarget(target *core.ContextTarget) *ContextTarget {
	loaded := &ContextTarget{
		Handle: target.Handle, SceneID: optionalID(target.SceneID), LayerID: optionalID(target.LayerID),
		BeatID: optionalID(target.BeatID), Field: target.Field, Mode: target.Mode,
		Kept: target.Kept, Override: target.Override, UpdatedAt: optionalTime(target.UpdatedAt),
		Sources: loadBeatRecapSources(target.Sources),
		Blocks: lo.Map(target.Blocks, func(block *core.ContextBlock, _ int) *ContextBlock {
			return &ContextBlock{Handle: block.Handle, Position: block.Position}
		}),
	}

	if target.Range != nil {
		loaded.Range = &Range{Start: target.Range.Start, End: target.Range.End}
	}

	return loaded
}

// requestRoles returns the roles the verified claims of a request carry, none without claims.
func requestRoles(r *http.Request) []string {
	// GetClaimsContext never fails: it returns nil claims for a request without them.
	claims, _ := serviceauthentication.GetClaimsContext(r.Context())
	if claims == nil {
		return nil
	}

	return claims.Roles
}
