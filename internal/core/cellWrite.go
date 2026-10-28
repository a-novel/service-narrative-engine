package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

// CellWriteDaoScene reads the cell's scene.
type CellWriteDaoScene interface {
	Exec(ctx context.Context, request *dao.SceneSelectRequest) (*dao.Scene, error)
}

// CellWriteDaoLayer reads the cell's layer, whose kind names the content schema.
type CellWriteDaoLayer interface {
	Exec(ctx context.Context, request *dao.LayerSelectRequest) (*dao.Layer, error)
}

// CellWriteDaoProject reads the cell's project, to check its owner.
type CellWriteDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// CellWriteDao writes the cell.
type CellWriteDao interface {
	Exec(ctx context.Context, request *dao.CellUpsertRequest) (*dao.Cell, error)
}

// CellWriteRequest carries a cell's new content.
type CellWriteRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	SceneID uuid.UUID `validate:"required"`
	LayerID uuid.UUID `validate:"required"`
	// Content is a JSON document that follows the content schema of the layer's kind.
	Content json.RawMessage `validate:"required"`
}

// CellWrite writes the content of a cell, which it validates against the content schema of the
// layer's kind. The first write creates the cell.
type CellWrite struct {
	daoScene   CellWriteDaoScene
	daoLayer   CellWriteDaoLayer
	daoProject CellWriteDaoProject
	dao        CellWriteDao
	config     config.Composer
}

func NewCellWrite(
	daoScene CellWriteDaoScene,
	daoLayer CellWriteDaoLayer,
	daoProject CellWriteDaoProject,
	dao CellWriteDao,
	config config.Composer,
) *CellWrite {
	return &CellWrite{daoScene: daoScene, daoLayer: daoLayer, daoProject: daoProject, dao: dao, config: config}
}

func (service *CellWrite) Exec(ctx context.Context, request *CellWriteRequest) (*Cell, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CellWrite")
	defer span.End()

	span.SetAttributes(
		attribute.String("cell.scene_id", request.SceneID.String()),
		attribute.String("cell.layer_id", request.LayerID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("cell.size", len(request.Content)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if len(request.Content) > service.config.MaxCellBytes {
		return nil, otel.ReportError(span, fmt.Errorf(
			"%w: %d bytes of content, at most %d", ErrInvalidRequest, len(request.Content), service.config.MaxCellBytes,
		))
	}

	scene, err := service.daoScene.Exec(ctx, &dao.SceneSelectRequest{ID: request.SceneID})
	if errors.Is(err, dao.ErrSceneSelectNotFound) {
		err = errors.Join(err, ErrSceneNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select scene: %w", err))
	}

	layer, err := service.daoLayer.Exec(ctx, &dao.LayerSelectRequest{ID: request.LayerID})
	if errors.Is(err, dao.ErrLayerSelectNotFound) {
		err = errors.Join(err, ErrLayerNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select layer: %w", err))
	}

	if scene.ProjectID != layer.ProjectID {
		return nil, otel.ReportError(span, ErrCellNotFound)
	}

	project, err := service.daoProject.Exec(ctx, &dao.ProjectSelectRequest{ID: scene.ProjectID})
	if errors.Is(err, dao.ErrProjectSelectNotFound) {
		err = errors.Join(err, ErrSceneNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select project: %w", err))
	}

	err = checkOwner(project, request.OwnerID, ErrSceneNotFound)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = validateContent(layer.Kind, request.Content)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("validate content: %w", err))
	}

	entity, err := service.dao.Exec(ctx, &dao.CellUpsertRequest{
		SceneID: request.SceneID,
		LayerID: request.LayerID,
		Content: request.Content,
		Now:     time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("write cell: %w", err))
	}

	return newCell(entity, 0), nil
}

// validateContent checks content against the content schema of a layer kind. Block IDs of a
// manuscript cell are unique as well, which a JSON Schema cannot express.
func validateContent(kind string, content json.RawMessage) error {
	schema, err := schemas.Content(kind)
	if err != nil {
		return fmt.Errorf("content schema: %w", err)
	}

	err = schema.Validate(content)
	if err != nil {
		return errors.Join(err, ErrInvalidRequest)
	}

	if kind != LayerKindManuscript {
		return nil
	}

	var manuscript struct {
		Blocks []struct {
			ID string `json:"id"`
		} `json:"blocks"`
	}

	err = json.Unmarshal(content, &manuscript)
	if err != nil {
		return errors.Join(err, ErrInvalidRequest)
	}

	seen := make(map[string]bool, len(manuscript.Blocks))
	for _, block := range manuscript.Blocks {
		if seen[block.ID] {
			return fmt.Errorf("%w: duplicate block id %q", ErrInvalidRequest, block.ID)
		}

		seen[block.ID] = true
	}

	return nil
}
