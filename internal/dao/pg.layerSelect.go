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

//go:embed pg.layerSelect.sql
var layerSelectQuery string

// ErrLayerSelectNotFound is returned by [PgLayerSelect.Exec] when no layer with the ID belongs to a
// project of the owner.
var ErrLayerSelectNotFound = errors.New("layer not found")

// LayerSelectRequest is the input to [PgLayerSelect.Exec].
type LayerSelectRequest struct {
	ID uuid.UUID
	// OwnerID is the user who must own the layer's project. Another owner's layer reads as not found.
	OwnerID uuid.UUID
}

// PgLayerSelect reads a layer of a project its owner holds.
type PgLayerSelect struct{}

func NewPgLayerSelect() *PgLayerSelect {
	return &PgLayerSelect{}
}

func (dao *PgLayerSelect) Exec(ctx context.Context, request *LayerSelectRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayerSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Layer{}

	err = db.NewRaw(layerSelectQuery, request.ID, request.OwnerID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrLayerSelectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
