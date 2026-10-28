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
	// ProjectID is the project whose cells to read.
	ProjectID uuid.UUID

	// Limit caps the cells in the page.
	Limit int
	// Offset is how many cells come before the page, in the order the page follows.
	Offset int
}

// CellListRow is one cell of a page.
type CellListRow struct {
	Cell

	// Total counts every cell of the project, beyond the page.
	Total int `bun:"total"`
}

// PgCellList reads a page of a project's cells, by the position of their scene, then of their
// layer. Each row carries the total, and a page past the end comes back empty.
type PgCellList struct{}

func NewPgCellList() *PgCellList {
	return &PgCellList{}
}

func (dao *PgCellList) Exec(ctx context.Context, request *CellListRequest) ([]*CellListRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCellList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*CellListRow

	err = db.NewRaw(cellListQuery, request.ProjectID, request.Limit, request.Offset).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("cells.count", len(entities)))

	return entities, nil
}
