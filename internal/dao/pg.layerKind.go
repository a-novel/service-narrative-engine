package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.layerKind.sql
var layerKindQuery string

// LayerKind is the kind of one layer, with how many layers of its project share it.
type LayerKind struct {
	ID uuid.UUID `bun:"id,type:uuid"`

	// Kind names the content schema the layer's cells follow.
	Kind string `bun:"kind"`
	// Count is how many layers of the project have the kind, this one included.
	Count int `bun:"count"`
}

// LayerKindRequest is the input to [PgLayerKind.Exec].
type LayerKindRequest struct {
	ProjectID uuid.UUID
	// IDs lists the layers to read.
	IDs []uuid.UUID
}

// PgLayerKind reads the kinds of layers of a project, in no particular order. It skips an ID the project
// does not hold.
type PgLayerKind struct{}

func NewPgLayerKind() *PgLayerKind {
	return &PgLayerKind{}
}

func (dao *PgLayerKind) Exec(ctx context.Context, request *LayerKindRequest) ([]*LayerKind, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayerKind")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*LayerKind

	err = db.NewRaw(layerKindQuery, request.ProjectID, bun.List(request.IDs), len(request.IDs)).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entities, nil
}
