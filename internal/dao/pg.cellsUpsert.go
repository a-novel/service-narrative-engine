package dao

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.cellsUpsert.sql
var cellsUpsertQuery string

var (
	// ErrCellsUpsertSceneNotFound is returned by [PgCellsUpsert.Exec] when the project holds no scene
	// with the ID.
	ErrCellsUpsertSceneNotFound = errors.New("scene not found")
	// ErrCellsUpsertLayerNotFound is returned by [PgCellsUpsert.Exec] when the project holds no layer
	// with the ID.
	ErrCellsUpsertLayerNotFound = errors.New("layer not found")
)

// CellsUpsertItem is one cell of a [CellsUpsertRequest].
type CellsUpsertItem struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID

	// Content is a JSON document that follows the content schema of the layer's kind.
	Content json.RawMessage
}

// CellsUpsertRequest is the input to [PgCellsUpsert.Exec].
type CellsUpsertRequest struct {
	ProjectID uuid.UUID
	// Cells lists the cells to write. A scene and a layer meet in at most one of them.
	Cells []CellsUpsertItem

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgCellsUpsert writes the content of cells, creating each cell on its first write, and returns the
// written cells in no particular order. Every scene and layer must belong to the project, or nothing
// is written.
type PgCellsUpsert struct{}

func NewPgCellsUpsert() *PgCellsUpsert {
	return &PgCellsUpsert{}
}

func (dao *PgCellsUpsert) Exec(ctx context.Context, request *CellsUpsertRequest) ([]*CellUpsertRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCellsUpsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	scenes := lo.Map(request.Cells, func(item CellsUpsertItem, _ int) uuid.UUID { return item.SceneID })
	layers := lo.Map(request.Cells, func(item CellsUpsertItem, _ int) uuid.UUID { return item.LayerID })
	contents := lo.Map(request.Cells, func(item CellsUpsertItem, _ int) string { return string(item.Content) })

	var entities []*CellUpsertRow

	err = db.NewRaw(
		cellsUpsertQuery, request.ProjectID, bun.List(scenes), bun.List(layers), bun.List(contents), request.Now,
	).Scan(ctx, &entities)

	switch {
	case isViolation(err, pgForeignKeyViolation, "cells_scene_fkey"):
		err = errors.Join(err, ErrCellsUpsertSceneNotFound)
	case isViolation(err, pgForeignKeyViolation, "cells_layer_fkey"):
		err = errors.Join(err, ErrCellsUpsertLayerNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entities, nil
}
