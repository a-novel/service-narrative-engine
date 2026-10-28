package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.cellCount.sql
var cellCountQuery string

// CellCountRequest is the input to [PgCellCount.Exec].
type CellCountRequest struct {
	// ProjectID is the project whose cells to count.
	ProjectID uuid.UUID
}

// CellCount is how many cells a project holds.
type CellCount struct {
	Count int `bun:"count"`
}

// PgCellCount counts the cells of a project, so a client paging through them knows how many to expect.
type PgCellCount struct{}

func NewPgCellCount() *PgCellCount {
	return &PgCellCount{}
}

func (dao *PgCellCount) Exec(ctx context.Context, request *CellCountRequest) (*CellCount, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCellCount")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &CellCount{}

	err = db.NewRaw(cellCountQuery, request.ProjectID).Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
