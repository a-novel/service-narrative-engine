package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.layerMove.sql
var layerMoveQuery string

// ErrLayerMoveNotFound is returned by [PgLayerMove.Exec] when the project holds no layer with the ID.
var ErrLayerMoveNotFound = errors.New("layer not found")

// LayerMoveRequest is the input to [PgLayerMove.Exec].
type LayerMoveRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID

	// Position is the layer's position after the move. A position past the last layer moves it last.
	Position int
}

// PgLayerMove moves a layer to another position, and shifts the layers between its old and new
// position by one toward the old one. The caller locks the project first.
type PgLayerMove struct{}

func NewPgLayerMove() *PgLayerMove {
	return &PgLayerMove{}
}

func (dao *PgLayerMove) Exec(ctx context.Context, request *LayerMoveRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayerMove")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Layer{}

	err = db.NewRaw(layerMoveQuery, request.ProjectID, request.ID, request.Position).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrLayerMoveNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
