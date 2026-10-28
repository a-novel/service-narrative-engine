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

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

// CellWriteDaoOwner checks that the user owns the project.
type CellWriteDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// CellWriteDaoKind reads the kind of the cell's layer, which names the content schema.
type CellWriteDaoKind interface {
	Exec(ctx context.Context, request *dao.LayerKindRequest) (string, error)
}

// CellWriteDao writes the cell. It refuses a scene of another project.
type CellWriteDao interface {
	Exec(ctx context.Context, request *dao.CellUpsertRequest) (*dao.Cell, error)
}

// CellWriteRequest carries a cell's new content.
type CellWriteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID `validate:"required"`
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID `validate:"required"`

	// Content is a JSON document that follows the content schema of the layer's kind, at most 2 MiB.
	Content json.RawMessage `validate:"required,max=2097152"`
}

// CellWrite writes the content of a cell, which it validates against the content schema of the
// layer's kind. The first write creates the cell.
type CellWrite struct {
	daoOwner CellWriteDaoOwner
	daoKind  CellWriteDaoKind
	dao      CellWriteDao
}

func NewCellWrite(daoOwner CellWriteDaoOwner, daoKind CellWriteDaoKind, dao CellWriteDao) *CellWrite {
	return &CellWrite{daoOwner: daoOwner, daoKind: daoKind, dao: dao}
}

func (service *CellWrite) Exec(ctx context.Context, request *CellWriteRequest) (*Cell, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CellWrite")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("cell.scene_id", request.SceneID.String()),
		attribute.String("cell.layer_id", request.LayerID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("cell.size", len(request.Content)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	kind, err := service.daoKind.Exec(ctx, &dao.LayerKindRequest{ProjectID: request.ProjectID, ID: request.LayerID})
	if errors.Is(err, dao.ErrLayerKindNotFound) {
		err = errors.Join(err, ErrLayerNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("read layer kind: %w", err))
	}

	schema, err := schemas.Content(kind)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("content schema: %w", err))
	}

	err = schema.Validate(request.Content)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.CellUpsertRequest{
		ProjectID: request.ProjectID,
		SceneID:   request.SceneID,
		LayerID:   request.LayerID,
		Content:   request.Content,
		Now:       time.Now(),
	})

	switch {
	case errors.Is(err, dao.ErrCellUpsertSceneNotFound):
		err = errors.Join(err, ErrSceneNotFound)
	case errors.Is(err, dao.ErrCellUpsertLayerNotFound):
		err = errors.Join(err, ErrLayerNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("write cell: %w", err))
	}

	return newCell(entity), nil
}
