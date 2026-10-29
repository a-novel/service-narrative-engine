package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

const (
	// GenerationKindSuggest writes targets: scene fields and cells.
	GenerationKindSuggest = "suggest"
	// GenerationKindConsolidate recaps one cell.
	GenerationKindConsolidate = "consolidate"
	// GenerationKindCheck looks for coherence issues across scenes.
	GenerationKindCheck = "check"
)

// GenerationKinds lists the kinds of generation. Each has an output schema, prompts in every project
// language, and a budget.
var GenerationKinds = []string{GenerationKindSuggest, GenerationKindConsolidate, GenerationKindCheck}

const (
	// TargetModeWrite writes a target anew. The model does not see the content it replaces.
	TargetModeWrite = "write"
	// TargetModeRevise changes a target from its current content, which the model sees.
	TargetModeRevise = "revise"
)

// TargetModes lists how a suggest generation may write a target.
var TargetModes = []string{TargetModeWrite, TargetModeRevise}

const (
	// SceneFieldTitle is the title of a scene.
	SceneFieldTitle = "title"
	// SceneFieldDescription is the description of a scene.
	SceneFieldDescription = "description"
)

// SceneFields lists the fields of a scene a suggest generation may write.
var SceneFields = []string{SceneFieldTitle, SceneFieldDescription}

// ErrContextTooLarge is returned by [GenerationPlan] when the frame, the task and the scenes of the targets
// alone outgrow the limit of the generation's kind. A narrower range or fewer targets fit.
var ErrContextTooLarge = errors.New("context too large")

var errKindNotConfigured = errors.New("generation kind not configured")

// Range is a span of positions, from Start up to End excluded. A range whose ends meet is an insertion
// point before Start.
type Range struct {
	Start int `validate:"min=0"`
	End   int `validate:"gtefield=Start"`
}

// GenerationTarget names one thing a suggest generation writes: a field of a scene, or a cell.
type GenerationTarget struct {
	SceneID uuid.UUID `validate:"required"`
	// LayerID names the cell of the scene in that layer. It is empty for a scene field.
	LayerID uuid.UUID `validate:"required_without=Field,excluded_with=Field"`
	// Field names the scene field to write, one of [SceneFields]. It is empty for a cell.
	Field string `validate:"omitempty,scenefield"`

	// Mode is how the generation writes the target, one of [TargetModes].
	Mode string `validate:"targetmode"`
	// Range narrows a manuscript target to the blocks the generation may change, by position. Nil covers
	// the whole cell.
	Range *Range
	// Kept lists the positions of blocks in the range that stay unchanged. The model sees them in both
	// modes.
	Kept []int `validate:"max=10000,unique,dive,min=0"`
}

// GenerationCell names the cell a consolidate generation recaps.
type GenerationCell struct {
	SceneID uuid.UUID `validate:"required"`
	LayerID uuid.UUID `validate:"required"`
}

// GenerationOverrides carries content the author has not saved. A generation reads it in place of the
// stored content, and stores none of it.
type GenerationOverrides struct {
	// Scenes replaces the fields of scenes, each scene once.
	Scenes []*SceneOverride `validate:"max=500,unique=ID,dive,required"`
	// Cells replaces the content of cells, each cell once.
	Cells []*CellOverride `validate:"max=100,dive,required"`
}

// SceneOverride is the unsaved text of a scene's fields.
type SceneOverride struct {
	ID uuid.UUID `validate:"required"`

	// Title names the scene in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Description says what the scene is for, written for the author.
	Description string `validate:"max=20000"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string `validate:"max=20000"`
}

// CellOverride is the unsaved content of a cell.
type CellOverride struct {
	SceneID uuid.UUID `validate:"required"`
	LayerID uuid.UUID `validate:"required"`

	// Content is a JSON document that follows the content schema of the layer's kind, at most 2 MiB.
	Content json.RawMessage `validate:"required,max=2097152"`
}

// GenerationPlanRequest asks for the context and the prompt of one generation of a project. Kind names
// the generation, and only the fields of that kind are set.
type GenerationPlanRequest struct {
	// Snapshot is the project the generation reads, as stored.
	Snapshot *ProjectSnapshot `validate:"required"`
	// Kind is the kind of the generation, one of [GenerationKinds].
	Kind string `validate:"generationkind"`

	// Targets lists what a suggest generation writes, from 1 to 20, each once.
	Targets []*GenerationTarget `validate:"excluded_unless=Kind suggest,max=20,dive,required"`
	// Cell names the cell a consolidate generation recaps, unless BeatID names a beat. A recap reads stored
	// content only.
	Cell *GenerationCell `validate:"excluded_unless=Kind consolidate"`
	// BeatID names the beat a consolidate generation recaps, from its description and its scenes'
	// manuscript.
	BeatID uuid.UUID `validate:"excluded_unless=Kind consolidate"`
	// Scenes narrows a check generation to a range of scene positions. Nil checks every scene.
	Scenes *Range `validate:"excluded_unless=Kind check"`

	// Instruction is what the author asks of the generation.
	Instruction string `validate:"max=4000,excluded_if=Kind consolidate"`
	// Exclude lists the layers the generation does not read. A target still shows its own cell.
	Exclude []uuid.UUID `validate:"max=1000,unique,excluded_if=Kind consolidate"`
	// Overrides carries unsaved content the generation reads in place of the stored one.
	Overrides *GenerationOverrides `validate:"excluded_if=Kind consolidate"`
}

// GenerationPlan plans what one generation reads and renders its prompt, in the project's language, with
// the model settings of its kind. Each scene enters at the richest level the budget of the kind allows, the
// targets' scenes always whole. The same request always gives the same plan and the same prompt.
type GenerationPlan struct {
	config config.Generations
}

func NewGenerationPlan(config config.Generations) *GenerationPlan {
	return &GenerationPlan{config: config}
}

func (service *GenerationPlan) Exec(ctx context.Context, request *GenerationPlanRequest) (*ContextPlan, error) {
	_, span := otel.Tracer().Start(ctx, "core.GenerationPlan")
	defer span.End()

	span.SetAttributes(attribute.String("generation.kind", request.Kind))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(attribute.String("project.id", request.Snapshot.Project.ID.String()))

	kind, ok := service.config.Kinds[request.Kind]
	if !ok {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %s", errKindNotConfigured, request.Kind))
	}

	draft, err := newContextDraft(request)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("read project: %w", err))
	}

	plan, err := draft.render(kind)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("render prompt: %w", err))
	}

	span.SetAttributes(attribute.Int("prompt.estimate", plan.Estimate))

	return plan, nil
}
