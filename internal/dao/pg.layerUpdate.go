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
// project of the owner at the revision.
var ErrLayerUpdateNotFound = errors.New("layer not found at this revision")

// LayerUpdateRequest is the input to [PgLayerUpdate.Exec].
type LayerUpdateRequest struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	// Revision is the revision the caller read.
	Revision    int
	Name        string
	Description string
	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgLayerUpdate replaces a layer's own fields when the stored revision is the one the caller read.
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
		layerUpdateQuery, request.ID, request.OwnerID, request.Revision, request.Name, request.Description, request.Now,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrLayerUpdateNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
