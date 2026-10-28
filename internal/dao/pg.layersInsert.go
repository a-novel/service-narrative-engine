package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.layersInsert.sql
var layersInsertQuery string

// ErrLayersInsertFull is returned by [PgLayersInsert.Exec] when a layer would take a position past
// 999, so the project would hold more than the 1000 layers the schema allows.
var ErrLayersInsertFull = errors.New("project holds the most layers it can")

// LayersInsertItem is one layer of a [LayersInsertRequest].
type LayersInsertItem struct {
	ID uuid.UUID

	// Kind names the content schema the layer's cells follow.
	Kind string

	// Name labels the layer's row in the composer.
	Name string
	// Description says what the layer records.
	Description string
}

// LayersInsertRequest is the input to [PgLayersInsert.Exec].
type LayersInsertRequest struct {
	// ProjectID is the project the layers join.
	ProjectID uuid.UUID

	// Position is the position of the first layer, and the others follow it in order. A position past the
	// last layer appends.
	Position int
	// Layers lists the layers to create, in order.
	Layers []LayersInsertItem

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgLayersInsert creates layers in one project from Position on, and moves the layers at or after
// Position down to make room. It returns the new layers in no particular order.
type PgLayersInsert struct{}

func NewPgLayersInsert() *PgLayersInsert {
	return &PgLayersInsert{}
}

func (dao *PgLayersInsert) Exec(ctx context.Context, request *LayersInsertRequest) ([]*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayersInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	ids := lo.Map(request.Layers, func(item LayersInsertItem, _ int) uuid.UUID { return item.ID })
	kinds := lo.Map(request.Layers, func(item LayersInsertItem, _ int) string { return item.Kind })
	names := lo.Map(request.Layers, func(item LayersInsertItem, _ int) string { return item.Name })
	descriptions := lo.Map(request.Layers, func(item LayersInsertItem, _ int) string { return item.Description })

	var entities []*Layer

	err = db.NewRaw(
		layersInsertQuery,
		request.ProjectID, request.Position, bun.List(ids), bun.List(kinds), bun.List(names),
		bun.List(descriptions), request.Now,
	).Scan(ctx, &entities)
	if isViolation(err, pgCheckViolation, "layers_position_check") {
		err = errors.Join(err, ErrLayersInsertFull)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("layers.count", len(entities)))

	return entities, nil
}
