package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.layerInsert.sql
var layerInsertQuery string

// ErrLayerInsertFull is returned by [PgLayerInsert.Exec] when the layer would take a position past
// 999, so the project would hold more than the 1000 layers the schema allows.
var ErrLayerInsertFull = errors.New("project holds the most layers it can")

// LayerInsertRequest is the input to [PgLayerInsert.Exec].
type LayerInsertRequest struct {
	ID uuid.UUID
	// ProjectID is the project the layer joins.
	ProjectID uuid.UUID

	// Position is the layer's place among the project's rows, from 0.
	Position int
	// Kind names the content schema the layer's cells follow.
	Kind string

	// Name labels the layer's row in the composer.
	Name string
	// Description says what the layer records.
	Description string

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgLayerInsert creates a layer at Position, and moves the layers at or after Position down to make
// room.
type PgLayerInsert struct{}

func NewPgLayerInsert() *PgLayerInsert {
	return &PgLayerInsert{}
}

func (dao *PgLayerInsert) Exec(ctx context.Context, request *LayerInsertRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayerInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Layer{}

	err = db.NewRaw(
		layerInsertQuery,
		request.ID, request.ProjectID, request.Position, request.Kind,
		request.Name, request.Description, request.Now,
	).Scan(ctx, entity)
	if isViolation(err, pgCheckViolation, "layers_position_check") {
		err = errors.Join(err, ErrLayerInsertFull)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
