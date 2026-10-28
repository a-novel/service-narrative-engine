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

//go:embed pg.sceneDelete.sql
var sceneDeleteQuery string

// SceneDeleteRequest is the input to [PgSceneDelete.Exec].
type SceneDeleteRequest struct {
	ProjectID uuid.UUID
	// IDs lists the scenes to delete.
	IDs []uuid.UUID
}

// PgSceneDelete removes scenes of a project with their cells, and moves the scenes after them up to close
// the gaps. It returns the deleted scenes, in no particular order, and skips an ID the project does not
// hold.
type PgSceneDelete struct{}

func NewPgSceneDelete() *PgSceneDelete {
	return &PgSceneDelete{}
}

func (dao *PgSceneDelete) Exec(ctx context.Context, request *SceneDeleteRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Scene

	err = db.NewRaw(sceneDeleteQuery, request.ProjectID, bun.List(request.IDs)).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("scenes.count", len(entities)))

	return entities, nil
}
