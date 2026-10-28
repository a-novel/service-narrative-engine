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

//go:embed pg.layerDelete.sql
var layerDeleteQuery string

// LayerDeleteRequest is the input to [PgLayerDelete.Exec].
type LayerDeleteRequest struct {
	ProjectID uuid.UUID
	// IDs lists the layers to delete.
	IDs []uuid.UUID
}

// PgLayerDelete removes layers of a project with their cells, and moves the layers after them up to close
// the gaps. It returns the deleted layers, in no particular order, and skips an ID the project does not
// hold.
type PgLayerDelete struct{}

func NewPgLayerDelete() *PgLayerDelete {
	return &PgLayerDelete{}
}

func (dao *PgLayerDelete) Exec(ctx context.Context, request *LayerDeleteRequest) ([]*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayerDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Layer

	err = db.NewRaw(layerDeleteQuery, request.ProjectID, bun.List(request.IDs)).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("layers.count", len(entities)))

	return entities, nil
}
