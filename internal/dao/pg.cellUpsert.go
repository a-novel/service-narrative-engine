package dao

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.cellUpsert.sql
var cellUpsertQuery string

// CellUpsertRequest is the input to [PgCellUpsert.Exec].
type CellUpsertRequest struct {
	SceneID uuid.UUID
	LayerID uuid.UUID
	Content json.RawMessage
	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgCellUpsert writes a cell's content, creating the cell on its first write.
type PgCellUpsert struct{}

func NewPgCellUpsert() *PgCellUpsert {
	return &PgCellUpsert{}
}

func (dao *PgCellUpsert) Exec(ctx context.Context, request *CellUpsertRequest) (*Cell, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCellUpsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Cell{}

	err = db.NewRaw(
		cellUpsertQuery, request.SceneID, request.LayerID, string(request.Content), request.Now,
	).Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
