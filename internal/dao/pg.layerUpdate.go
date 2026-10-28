package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.layerUpdate.sql
var layerUpdateQuery string

// ErrLayerUpdateNotFound is returned by [PgLayerUpdate.Exec] when no layer with the ID belongs to a
// project of the owner.
var ErrLayerUpdateNotFound = errors.New("layer not found")

// LayerUpdateRequest is the input to [PgLayerUpdate.Exec].
type LayerUpdateRequest struct {
	ID uuid.UUID
	// OwnerID is the user who must own the layer's project. Another owner's layer reads as not found.
	OwnerID uuid.UUID

	// Name labels the layer's row in the composer.
	Name string
	// Description says what the layer records.
	Description string

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgLayerUpdate replaces a layer's own fields.
type PgLayerUpdate struct{}

func NewPgLayerUpdate() *PgLayerUpdate {
	return &PgLayerUpdate{}
}

func (dao *PgLayerUpdate) Exec(ctx context.Context, request *LayerUpdateRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayerUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Layer{}

	err = db.NewRaw(
		layerUpdateQuery, request.ID, request.OwnerID, request.Name, request.Description, request.Now,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrLayerUpdateNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
