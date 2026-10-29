package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/samber/lo"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestGenerationSubmitService is the core operation [RestGenerationSubmit] delegates to.
type RestGenerationSubmitService interface {
	Exec(ctx context.Context, request *core.GenerationSubmitRequest) (*core.Generation, error)
}

// RestGenerationSubmitRequest is the JSON body the submit-generation endpoint accepts: a suggest
// generation on a project of the caller.
type RestGenerationSubmitRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Targets lists what the generation writes, from 1 to 20, each once.
	Targets []*RestGenerationTarget `json:"targets"`
	// Instruction is what the author asks of the generation.
	Instruction string `json:"instruction"`
	// Exclude lists the layers the generation does not read. A target still shows its own cell.
	Exclude []uuid.UUID `json:"exclude"`
	// Overrides carries unsaved content the generation reads in place of the stored one.
	Overrides *RestGenerationOverrides `json:"overrides"`

	// Variant asks for another result of the same calls.
	Variant uint32 `json:"variant"`
}

// RestGenerationTarget names one thing a suggest generation writes: a field of a scene, or a cell.
type RestGenerationTarget struct {
	SceneID uuid.UUID `json:"sceneID"`
	// LayerID names the cell of the scene in that layer. It is empty for a scene field.
	LayerID uuid.UUID `json:"layerID"`
	// Field names the scene field to write: title or description. It is empty for a cell.
	Field string `json:"field"`

	// Mode is how the generation writes the target: write or revise.
	Mode string `json:"mode"`
	// Range narrows a manuscript target to the blocks the generation may change. Null covers the cell.
	Range *RestRange `json:"range"`
	// Kept lists the positions of blocks in the range that stay unchanged.
	Kept []int `json:"kept"`
}

// RestGenerationOverrides carries content the author has not saved.
type RestGenerationOverrides struct {
	// Scenes replaces the fields of scenes, each scene once.
	Scenes []*RestSceneOverride `json:"scenes"`
	// Cells replaces the content of cells, each cell once.
	Cells []*RestCellOverride `json:"cells"`
}

// RestSceneOverride is the unsaved text of a scene's fields.
type RestSceneOverride struct {
	ID uuid.UUID `json:"id"`

	Title       string `json:"title"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
}

// RestCellOverride is the unsaved content of a cell.
type RestCellOverride struct {
	SceneID uuid.UUID `json:"sceneID"`
	LayerID uuid.UUID `json:"layerID"`

	// Content is a JSON document that follows the content schema of the layer's kind.
	Content json.RawMessage `json:"content"`
}

// RestGenerationSubmit is the HTTP handler for the submit-generation endpoint. It responds 202 with each
// call of the generation, and 429 with the cap genai refused it for when genai admits no call. A caller
// granted [config.PermissionGenerationUncapped] runs on genai's admin tier.
type RestGenerationSubmit struct {
	service RestGenerationSubmitService
	// uncapped holds the roles that grant [config.PermissionGenerationUncapped].
	uncapped map[string]bool
	logger   logging.Log
}

func NewRestGenerationSubmit(
	service RestGenerationSubmitService, permissions serviceauthentication.Permissions, logger logging.Log,
) *RestGenerationSubmit {
	return &RestGenerationSubmit{
		service:  service,
		uncapped: rolesGranting(permissions, config.PermissionGenerationUncapped),
		logger:   logger,
	}
}

func (handler *RestGenerationSubmit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.GenerationSubmit")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestGenerationSubmitRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.GenerationSubmitRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Admin:     requestGranted(r, handler.uncapped),
		Targets: lo.Map(request.Targets, func(target *RestGenerationTarget, _ int) *core.GenerationTarget {
			return saveGenerationTarget(target)
		}),
		Instruction: request.Instruction,
		Exclude:     request.Exclude,
		Overrides:   saveGenerationOverrides(request.Overrides),
		Variant:     request.Variant,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)

	var quota *core.GenerationQuotaError
	if errors.As(err, &quota) {
		err = httpf.WithTag(err, "cap", quota.Cap)
	}

	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, generationErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusAccepted, loadGeneration(result))
}

// saveGenerationTarget returns nil for a missing target, which the core refuses.
func saveGenerationTarget(target *RestGenerationTarget) *core.GenerationTarget {
	if target == nil {
		return nil
	}

	saved := &core.GenerationTarget{
		SceneID: target.SceneID, LayerID: target.LayerID, Field: target.Field, Mode: target.Mode, Kept: target.Kept,
	}

	if target.Range != nil {
		saved.Range = &core.Range{Start: target.Range.Start, End: target.Range.End}
	}

	return saved
}

func saveGenerationOverrides(overrides *RestGenerationOverrides) *core.GenerationOverrides {
	if overrides == nil {
		return nil
	}

	return &core.GenerationOverrides{
		Scenes: lo.Map(overrides.Scenes, func(scene *RestSceneOverride, _ int) *core.SceneOverride {
			if scene == nil {
				return nil
			}

			return &core.SceneOverride{
				ID: scene.ID, Title: scene.Title, Description: scene.Description, Prompt: scene.Prompt,
			}
		}),
		Cells: lo.Map(overrides.Cells, func(cell *RestCellOverride, _ int) *core.CellOverride {
			if cell == nil {
				return nil
			}

			return &core.CellOverride{SceneID: cell.SceneID, LayerID: cell.LayerID, Content: cell.Content}
		}),
	}
}
