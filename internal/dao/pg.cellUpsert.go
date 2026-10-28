package dao

import (
	"context"
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

var (
	// ErrCellUpsertSceneNotFound is returned by [PgCellUpsert.Exec] when the project holds no scene
	// with the ID.
	ErrCellUpsertSceneNotFound = errors.New("scene not found")
	// ErrCellUpsertLayerNotFound is returned by [PgCellUpsert.Exec] when the project holds no layer
	// with the ID.
	ErrCellUpsertLayerNotFound = errors.New("layer not found")
)

// CellUpsertRequest is the input to [PgCellUpsert.Exec].
type CellUpsertRequest struct {
	ProjectID uuid.UUID
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID

	// Content is a JSON document that follows the content schema of the layer's kind.
	Content json.RawMessage

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgCellUpsert writes a cell's content, creating the cell on its first write. The scene and the
// layer must belong to the project.
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
		cellUpsertQuery, request.ProjectID, request.SceneID, request.LayerID, string(request.Content), request.Now,
	).Scan(ctx, entity)

	switch {
	case isViolation(err, pgForeignKeyViolation, "cells_scene_fkey"):
		err = errors.Join(err, ErrCellUpsertSceneNotFound)
	case isViolation(err, pgForeignKeyViolation, "cells_layer_fkey"):
		err = errors.Join(err, ErrCellUpsertLayerNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
