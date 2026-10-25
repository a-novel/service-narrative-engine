package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.cellUpsert.sql
var cellUpsertQuery string

// ErrCellUpsertNotFound is returned by [PgCellUpsert.Exec] when the cell is not at the revision, or
// already exists for revision 0.
var ErrCellUpsertNotFound = errors.New("cell not found at this revision")

// CellUpsertRequest is the input to [PgCellUpsert.Exec].
type CellUpsertRequest struct {
	SceneID uuid.UUID
	LayerID uuid.UUID
	// Revision is the revision the caller read, 0 for a cell it has never seen.
	Revision int
	Content  json.RawMessage
	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgCellUpsert writes a cell's content when the stored revision is the one the caller read. Revision 0
// creates a cell that does not exist yet.
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
		cellUpsertQuery, request.SceneID, request.LayerID, request.Revision, string(request.Content), request.Now,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrCellUpsertNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
