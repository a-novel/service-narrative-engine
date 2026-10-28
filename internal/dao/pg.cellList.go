package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.cellList.sql
var cellListQuery string

// CellListRequest is the input to [PgCellList.Exec].
type CellListRequest struct {
	ProjectID uuid.UUID
}

// PgCellList reads the cells of a project, in the narrative order of their scenes.
type PgCellList struct{}

func NewPgCellList() *PgCellList {
	return &PgCellList{}
}

func (dao *PgCellList) Exec(ctx context.Context, request *CellListRequest) ([]*Cell, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCellList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Cell

	err = db.NewRaw(cellListQuery, request.ProjectID).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("cells.count", len(entities)))

	return entities, nil
}
