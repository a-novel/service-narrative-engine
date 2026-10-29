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

// CellRecapWriteDaoOwner checks that the user owns the project.
type CellRecapWriteDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// CellRecapWriteDao writes the recap of a cell of the project.
type CellRecapWriteDao interface {
	Exec(ctx context.Context, request *dao.CellRecapUpdateRequest) error
}

// CellRecapWriteRequest carries the recap a consolidation wrote for a cell.
type CellRecapWriteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID `validate:"required"`
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID `validate:"required"`

	// Recap is the output of the consolidation, a JSON document that follows the consolidate output schema.
	Recap json.RawMessage `validate:"required,max=2097152"`
	// CellUpdatedAt is the cell's UpdatedAt the consolidation read, as its context reports it.
	CellUpdatedAt time.Time `validate:"required"`
}

// CellRecapWrite stores the recap a consolidation wrote for a cell of the user's project, stamped with the
// version of the cell it read, and leaves the cell's UpdatedAt as it is. It refuses a recap that read the
// cell at a version older than the stored recap's, or newer than the cell's own.
type CellRecapWrite struct {
	daoOwner CellRecapWriteDaoOwner
	dao      CellRecapWriteDao
}

func NewCellRecapWrite(daoOwner CellRecapWriteDaoOwner, dao CellRecapWriteDao) *CellRecapWrite {
	return &CellRecapWrite{daoOwner: daoOwner, dao: dao}
}

func (service *CellRecapWrite) Exec(ctx context.Context, request *CellRecapWriteRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "core.CellRecapWrite")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("cell.scene_id", request.SceneID.String()),
		attribute.String("cell.layer_id", request.LayerID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validateRecap(request, request.Recap)
	if err != nil {
		return otel.ReportError(span, err)
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	err = service.dao.Exec(ctx, &dao.CellRecapUpdateRequest{
		ProjectID:     request.ProjectID,
		SceneID:       request.SceneID,
		LayerID:       request.LayerID,
		Recap:         request.Recap,
		CellUpdatedAt: request.CellUpdatedAt,
	})

	switch {
	case errors.Is(err, dao.ErrCellRecapUpdateNotFound):
		err = errors.Join(err, ErrCellNotFound)
	case errors.Is(err, dao.ErrCellRecapUpdateConflict):
		err = errors.Join(err, ErrRecapConflict)
	}

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("write recap: %w", err))
	}

	return nil
}

// validateRecap checks a recap write request, and its recap against the consolidate output schema.
func validateRecap(request any, recap json.RawMessage) error {
	err := validate.Struct(request)
	if err != nil {
		return errors.Join(err, ErrInvalidRequest)
	}

	schema, err := schemas.Output(GenerationKindConsolidate)
	if err != nil {
		return fmt.Errorf("output schema: %w", err)
	}

	err = schema.Validate(recap)
	if err != nil {
		return errors.Join(err, ErrInvalidRequest)
	}

	return nil
}
