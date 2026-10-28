package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.layersUpdate.sql
var layersUpdateQuery string

// LayersUpdateItem is one layer of a [LayersUpdateRequest].
type LayersUpdateItem struct {
	ID uuid.UUID

	// Name labels the layer's row in the composer.
	Name string
	// Description says what the layer records.
	Description string
}

// LayersUpdateRequest is the input to [PgLayersUpdate.Exec].
type LayersUpdateRequest struct {
	ProjectID uuid.UUID
	// Layers lists the layers to update, each once.
	Layers []LayersUpdateItem

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgLayersUpdate replaces the own fields of layers of a project, and returns them in no particular order. It
// updates all of them, or none when the project does not hold one of them.
type PgLayersUpdate struct{}

func NewPgLayersUpdate() *PgLayersUpdate {
	return &PgLayersUpdate{}
}

func (dao *PgLayersUpdate) Exec(ctx context.Context, request *LayersUpdateRequest) ([]*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayersUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	ids := lo.Map(request.Layers, func(item LayersUpdateItem, _ int) uuid.UUID { return item.ID })
	names := lo.Map(request.Layers, func(item LayersUpdateItem, _ int) string { return item.Name })
	descriptions := lo.Map(request.Layers, func(item LayersUpdateItem, _ int) string { return item.Description })

	var entities []*Layer

	err = db.NewRaw(
		layersUpdateQuery, request.ProjectID, bun.List(ids), bun.List(names), bun.List(descriptions), request.Now,
	).
		Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("layers.count", len(entities)))

	return entities, nil
}
