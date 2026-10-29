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

//go:embed pg.cellRecapUpdate.sql
var cellRecapUpdateQuery string

var (
	// ErrCellRecapUpdateNotFound is returned by [PgCellRecapUpdate.Exec] when the project holds no cell
	// where the scene and the layer cross.
	ErrCellRecapUpdateNotFound = errors.New("cell not found")
	// ErrCellRecapUpdateConflict is returned by [PgCellRecapUpdate.Exec] when the recap read a version of the
	// cell older than the stored recap's, or newer than the cell's own.
	ErrCellRecapUpdateConflict = errors.New("recap stamp refused")
)

// CellRecapUpdateRequest is the input to [PgCellRecapUpdate.Exec].
type CellRecapUpdateRequest struct {
	ProjectID uuid.UUID
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID

	// Recap is a JSON document that follows the consolidate output schema.
	Recap json.RawMessage
	// CellUpdatedAt is the cell's UpdatedAt the recap read.
	CellUpdatedAt time.Time
}

// PgCellRecapUpdate writes the recap of a cell of the project, leaving the cell's UpdatedAt as it is.
type PgCellRecapUpdate struct{}

func NewPgCellRecapUpdate() *PgCellRecapUpdate {
	return &PgCellRecapUpdate{}
}

func (dao *PgCellRecapUpdate) Exec(ctx context.Context, request *CellRecapUpdateRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCellRecapUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var written bool

	err = db.NewRaw(
		cellRecapUpdateQuery,
		request.ProjectID, request.SceneID, request.LayerID, string(request.Recap), request.CellUpdatedAt,
	).Scan(ctx, &written)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		err = errors.Join(err, ErrCellRecapUpdateNotFound)
	case err == nil && !written:
		err = ErrCellRecapUpdateConflict
	}

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
