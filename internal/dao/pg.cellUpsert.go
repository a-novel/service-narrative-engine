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

//go:embed pg.cellUpsert.sql
var cellUpsertQuery string

var (
	// ErrCellUpsertSceneNotFound is returned by [PgCellUpsert.Exec] when the project holds no scene
	// with the ID.
	ErrCellUpsertSceneNotFound = errors.New("scene not found")
	// ErrCellUpsertLayerNotFound is returned by [PgCellUpsert.Exec] when the project holds no layer
	// with the ID.
	ErrCellUpsertLayerNotFound = errors.New("layer not found")
)

// CellUpsertItem is one cell of a [CellUpsertRequest].
type CellUpsertItem struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID

	// Content is a JSON document that follows the content schema of the layer's kind.
	Content json.RawMessage
}

// CellUpsertRequest is the input to [PgCellUpsert.Exec].
type CellUpsertRequest struct {
	ProjectID uuid.UUID
	// Cells lists the cells to write. A scene and a layer meet in at most one of them.
	Cells []CellUpsertItem

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgCellUpsert writes the content of cells, creating each cell on its first write, and returns them in
// no particular order. Every scene and layer must belong to the project, or nothing is written.
type PgCellUpsert struct{}

func NewPgCellUpsert() *PgCellUpsert {
	return &PgCellUpsert{}
}

func (dao *PgCellUpsert) Exec(ctx context.Context, request *CellUpsertRequest) ([]*Cell, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCellUpsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	scenes := lo.Map(request.Cells, func(item CellUpsertItem, _ int) uuid.UUID { return item.SceneID })
	layers := lo.Map(request.Cells, func(item CellUpsertItem, _ int) uuid.UUID { return item.LayerID })
	contents := lo.Map(request.Cells, func(item CellUpsertItem, _ int) string { return string(item.Content) })

	var entities []*Cell

	err = db.NewRaw(
		cellUpsertQuery, request.ProjectID, bun.List(scenes), bun.List(layers), bun.List(contents), request.Now,
	).Scan(ctx, &entities)

	switch {
	case isViolation(err, pgForeignKeyViolation, "cells_scene_fkey"):
		err = errors.Join(err, ErrCellUpsertSceneNotFound)
	case isViolation(err, pgForeignKeyViolation, "cells_layer_fkey"):
		err = errors.Join(err, ErrCellUpsertLayerNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entities, nil
}
