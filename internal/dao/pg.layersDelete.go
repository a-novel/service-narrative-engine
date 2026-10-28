package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.layersDelete.sql
var layersDeleteQuery string

// LayersDeleteRequest is the input to [PgLayersDelete.Exec].
type LayersDeleteRequest struct {
	ProjectID uuid.UUID
	// IDs lists the layers to delete.
	IDs []uuid.UUID
}

// PgLayersDelete removes layers of a project with their cells, and moves the layers after them up to close
// the gaps. It returns the deleted layers, in no particular order, and skips an ID the project does not
// hold.
type PgLayersDelete struct{}

func NewPgLayersDelete() *PgLayersDelete {
	return &PgLayersDelete{}
}

func (dao *PgLayersDelete) Exec(ctx context.Context, request *LayersDeleteRequest) ([]*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayersDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Layer

	err = db.NewRaw(layersDeleteQuery, request.ProjectID, bun.List(request.IDs)).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("layers.count", len(entities)))

	return entities, nil
}
