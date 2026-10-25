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

//go:embed pg.sceneReorder.sql
var sceneReorderQuery string

// SceneReorderRequest is the input to [PgSceneReorder.Exec].
type SceneReorderRequest struct {
	ProjectID uuid.UUID
	// IDs lists every scene of the project, in their new order.
	IDs []uuid.UUID
}

// PgSceneReorder sets the positions of a project's scenes to the order of IDs. It returns the scenes
// it moved, in no particular order.
type PgSceneReorder struct{}

func NewPgSceneReorder() *PgSceneReorder {
	return &PgSceneReorder{}
}

func (dao *PgSceneReorder) Exec(ctx context.Context, request *SceneReorderRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneReorder")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Scene

	err = db.NewRaw(sceneReorderQuery, request.ProjectID, bun.List(request.IDs)).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("scenes.count", len(entities)))

	return entities, nil
}
