package core

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

const (
	// ContextLevelFull shows every cell of a scene the generation reads.
	ContextLevelFull = "full"
	// ContextLevelRecap shows the fresh recap of a scene's manuscript in place of the manuscript, and its
	// other cells whole.
	ContextLevelRecap = "recap"
	// ContextLevelOutline shows the cells of a scene other than its manuscript.
	ContextLevelOutline = "outline"
	// ContextLevelOmitted leaves a scene out of the neighborhood. The frame still lists it.
	ContextLevelOmitted = "omitted"
)

// ContextPlan says what one generation reads and how the model names it. The model sees short handles,
// such as S1 for a scene or B3 for a block, and the plan maps each back to what it names. A handle the
// plan does not hold was never issued.
type ContextPlan struct {
	// Kind is the kind of the generation, one of [GenerationKinds].
	Kind string

	// Scenes lists every scene of the project, in narrative order, with the level it enters at.
	Scenes []*ContextScene
	// Layers lists the layers the context shows, in order.
	Layers []*ContextLayer
	// Targets lists what the generation writes or recaps, in the order of the request.
	Targets []*ContextTarget

	// Budget caps the token estimate of the prompt, from the configuration of the kind.
	Budget int
	// Estimate is the token estimate of the prompt.
	Estimate int

	// Prompt is what the model receives.
	Prompt *Prompt
}

// Scene returns the scene the model names by handle, such as S1.
func (plan *ContextPlan) Scene(handle string) (*ContextScene, bool) {
	return lo.Find(plan.Scenes, func(scene *ContextScene) bool { return scene.Handle == handle })
}

// Target returns the target the model names by handle, such as T1.
func (plan *ContextPlan) Target(handle string) (*ContextTarget, bool) {
	return lo.Find(plan.Targets, func(target *ContextTarget) bool { return target.Handle == handle })
}

// ContextScene is one scene as a generation reads it.
type ContextScene struct {
	Handle  string
	SceneID uuid.UUID

	// Level is how much of the scene the neighborhood shows: [ContextLevelFull], [ContextLevelRecap],
	// [ContextLevelOutline] or [ContextLevelOmitted]. A scene a generation writes in is always full.
	Level string

	// Override marks a scene whose fields come from the request.
	Override bool
	// Cells lists the cells the neighborhood shows for the scene, by layer order. A target's own cell
	// shows in the task, and the plan lists it with the target.
	Cells []*ContextCell

	// UpdatedAt is when the stored scene was written. It is zero for an override.
	UpdatedAt time.Time
}

// ContextCell is one cell a context shows.
type ContextCell struct {
	LayerID uuid.UUID

	// Recap marks a cell shown as its recap.
	Recap bool
	// Override marks a cell whose content comes from the request.
	Override bool

	// UpdatedAt is when the content the context read was written. It is zero for an override.
	UpdatedAt time.Time
}

// ContextLayer is one layer a context shows: a layer the request reads, or the layer of a target.
type ContextLayer struct {
	Handle  string
	LayerID uuid.UUID
}

// ContextTarget is one target of a generation, with the handles of the blocks the model may name.
type ContextTarget struct {
	Handle string

	SceneID uuid.UUID
	// LayerID names the target's cell. It is empty for a scene field.
	LayerID uuid.UUID
	// Field names the target's scene field, one of [SceneFields]. It is empty for a cell.
	Field string

	// Mode is how the generation writes the target, one of [TargetModes]. It is empty for a recap.
	Mode string
	// Range is the span of a manuscript target's blocks the generation may change.
	Range *Range
	// Kept lists the positions of the blocks in the range that stay unchanged.
	Kept []int
	// Blocks maps the handle of each block the target shows inside its range to the block's position.
	Blocks []*ContextBlock

	// Override marks a target whose current content comes from the request.
	Override bool
	// UpdatedAt is when the target's stored content was written. It is zero for an override, and for a
	// cell never written.
	UpdatedAt time.Time
}

// Block returns the position of the block the model names by handle, such as B3, among the blocks the
// target shows inside its range.
func (target *ContextTarget) Block(handle string) (int, bool) {
	block, ok := lo.Find(target.Blocks, func(block *ContextBlock) bool { return block.Handle == handle })
	if !ok {
		return 0, false
	}

	return block.Position, true
}

// ContextBlock is a block a target shows the model under a handle.
type ContextBlock struct {
	Handle string
	// Position is the block's position in the target's cell, from 0.
	Position int
}

// Prompt is what a model receives for one generation, laid out most stable first, so the start of the
// prompt caches across a project's generations.
type Prompt struct {
	// Developer is the developer message of the generation's kind, in the project's language.
	Developer string
	// Schema is the output schema of the kind, in the subset of JSON Schema strict structured outputs
	// accept.
	Schema json.RawMessage
	// User holds the frame, the neighborhood and the task, in XML elements.
	User string
}
