package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

// CellWriteDaoOwner checks that the user owns the project.
type CellWriteDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// CellWriteDaoKind reads the kinds of the cells' layers, which name their content schemas.
type CellWriteDaoKind interface {
	Exec(ctx context.Context, request *dao.LayerKindRequest) ([]*dao.LayerKind, error)
}

// CellWriteDao writes the cells. It refuses a scene or a layer of another project.
type CellWriteDao interface {
	Exec(ctx context.Context, request *dao.CellUpsertRequest) ([]*dao.Cell, error)
}

// CellWriteItem is one cell of a [CellWriteRequest].
type CellWriteItem struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID `validate:"required"`
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID `validate:"required"`

	// Content is a JSON document that follows the content schema of the layer's kind, at most 2 MiB.
	Content json.RawMessage `validate:"required,max=2097152"`
}

// CellWriteRequest carries the new content of cells.
type CellWriteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// Cells lists the cells to write, at most 100. A scene and a layer meet in at most one of them.
	Cells []CellWriteItem `validate:"required,min=1,max=100,uniquecells,dive"`
}

// CellWrite writes the content of cells, which it validates against the content schema of each
// layer's kind, and returns them in the order of the request. The first write of a cell creates it.
// It writes all of them or none.
type CellWrite struct {
	daoOwner CellWriteDaoOwner
	daoKind  CellWriteDaoKind
	dao      CellWriteDao
}

func NewCellWrite(daoOwner CellWriteDaoOwner, daoKind CellWriteDaoKind, dao CellWriteDao) *CellWrite {
	return &CellWrite{daoOwner: daoOwner, daoKind: daoKind, dao: dao}
}

func (service *CellWrite) Exec(ctx context.Context, request *CellWriteRequest) ([]*Cell, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CellWrite")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("cells.count", len(request.Cells)),
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

	layerIDs := lo.Uniq(lo.Map(request.Cells, func(cell CellWriteItem, _ int) uuid.UUID { return cell.LayerID }))

	kinds, err := service.daoKind.Exec(ctx, &dao.LayerKindRequest{ProjectID: request.ProjectID, IDs: layerIDs})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("read layer kinds: %w", err))
	}

	if len(kinds) != len(layerIDs) {
		return nil, otel.ReportError(span, fmt.Errorf(
			"%w: %d of %d layers", ErrLayerNotFound, len(layerIDs)-len(kinds), len(layerIDs),
		))
	}

	kindOf := lo.SliceToMap(kinds, func(kind *dao.LayerKind) (uuid.UUID, string) { return kind.ID, kind.Kind })

	for index, cell := range request.Cells {
		schema, err := schemas.Content(kindOf[cell.LayerID])
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("content schema: %w", err))
		}

		err = schema.Validate(cell.Content)
		if err != nil {
			return nil, otel.ReportError(span, errors.Join(fmt.Errorf("cell %d: %w", index, err), ErrInvalidRequest))
		}
	}

	entities, err := service.dao.Exec(ctx, &dao.CellUpsertRequest{
		ProjectID: request.ProjectID,
		Cells: lo.Map(request.Cells, func(cell CellWriteItem, _ int) dao.CellUpsertItem {
			return dao.CellUpsertItem{SceneID: cell.SceneID, LayerID: cell.LayerID, Content: cell.Content}
		}),
		Now: time.Now(),
	})

	switch {
	case errors.Is(err, dao.ErrCellUpsertSceneNotFound):
		err = errors.Join(err, ErrSceneNotFound)
	case errors.Is(err, dao.ErrCellUpsertLayerNotFound):
		err = errors.Join(err, ErrLayerNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("write cells: %w", err))
	}

	written := lo.KeyBy(entities, func(entity *dao.Cell) [2]uuid.UUID {
		return [2]uuid.UUID{entity.SceneID, entity.LayerID}
	})

	return lo.Map(request.Cells, func(cell CellWriteItem, _ int) *Cell {
		return newCell(written[[2]uuid.UUID{cell.SceneID, cell.LayerID}])
	}), nil
}
