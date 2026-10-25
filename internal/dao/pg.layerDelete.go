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

//go:embed pg.layerDelete.sql
var layerDeleteQuery string

// ErrLayerDeleteNotFound is returned by [PgLayerDelete.Exec] when no layer with the ID belongs to a
// project of the owner at the revision.
var ErrLayerDeleteNotFound = errors.New("layer not found at this revision")

// LayerDeleteRequest is the input to [PgLayerDelete.Exec].
type LayerDeleteRequest struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	// Revision is the revision the caller read.
	Revision int
}

// PgLayerDelete removes a layer with its cells when the stored revision is the one the caller read.
type PgLayerDelete struct{}

func NewPgLayerDelete() *PgLayerDelete {
	return &PgLayerDelete{}
}

func (dao *PgLayerDelete) Exec(ctx context.Context, request *LayerDeleteRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayerDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Layer{}

	err = db.NewRaw(layerDeleteQuery, request.ID, request.OwnerID, request.Revision).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrLayerDeleteNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
