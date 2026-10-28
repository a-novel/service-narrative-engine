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

// CellsWriteDaoOwner checks that the user owns the project.
type CellsWriteDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// CellsWriteDaoKind reads the kinds of the cells' layers, which name their content schemas.
type CellsWriteDaoKind interface {
	Exec(ctx context.Context, request *dao.LayerKindRequest) ([]*dao.LayerKind, error)
}

// CellsWriteDao writes the cells. It refuses a scene or a layer of another project.
type CellsWriteDao interface {
	Exec(ctx context.Context, request *dao.CellsUpsertRequest) ([]*dao.CellUpsertRow, error)
}

// CellsWriteItem is one cell of a [CellsWriteRequest].
type CellsWriteItem struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID `validate:"required"`
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID `validate:"required"`

	// Content is a JSON document that follows the content schema of the layer's kind, at most 2 MiB.
	Content json.RawMessage `validate:"required,max=2097152"`
}

// CellsWriteRequest carries the new content of cells.
type CellsWriteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// Cells lists the cells to write, at most 100. A scene and a layer meet in at most one of them.
	Cells []CellsWriteItem `validate:"required,min=1,max=100,uniquecells,dive"`
}

// CellsWrite writes the content of cells, which it validates against the content schema of each
// layer's kind, and returns the written cells in the order of the request, without their content. The
// first write of a cell creates it, and it writes all of them or none.
type CellsWrite struct {
	daoOwner CellsWriteDaoOwner
	daoKind  CellsWriteDaoKind
	dao      CellsWriteDao
}

func NewCellsWrite(daoOwner CellsWriteDaoOwner, daoKind CellsWriteDaoKind, dao CellsWriteDao) *CellsWrite {
	return &CellsWrite{daoOwner: daoOwner, daoKind: daoKind, dao: dao}
}

func (service *CellsWrite) Exec(ctx context.Context, request *CellsWriteRequest) ([]*WrittenCell, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CellsWrite")
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

	layerIDs := lo.Uniq(lo.Map(request.Cells, func(cell CellsWriteItem, _ int) uuid.UUID { return cell.LayerID }))

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

	entities, err := service.dao.Exec(ctx, &dao.CellsUpsertRequest{
		ProjectID: request.ProjectID,
		Cells: lo.Map(request.Cells, func(cell CellsWriteItem, _ int) dao.CellsUpsertItem {
			return dao.CellsUpsertItem{SceneID: cell.SceneID, LayerID: cell.LayerID, Content: cell.Content}
		}),
		Now: time.Now(),
	})

	switch {
	case errors.Is(err, dao.ErrCellsUpsertSceneNotFound):
		err = errors.Join(err, ErrSceneNotFound)
	case errors.Is(err, dao.ErrCellsUpsertLayerNotFound):
		err = errors.Join(err, ErrLayerNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("write cells: %w", err))
	}

	written := lo.KeyBy(entities, func(row *dao.CellUpsertRow) [2]uuid.UUID {
		return [2]uuid.UUID{row.SceneID, row.LayerID}
	})

	return lo.Map(request.Cells, func(cell CellsWriteItem, _ int) *WrittenCell {
		return newWrittenCell(written[[2]uuid.UUID{cell.SceneID, cell.LayerID}])
	}), nil
}
