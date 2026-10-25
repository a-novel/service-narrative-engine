package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sceneList.sql
var sceneListQuery string

// SceneListRequest is the input to [PgSceneList.Exec].
type SceneListRequest struct {
	ProjectID uuid.UUID
	Limit     int
}

// PgSceneList reads the scenes of a project in narrative order.
type PgSceneList struct{}

func NewPgSceneList() *PgSceneList {
	return &PgSceneList{}
}

func (dao *PgSceneList) Exec(ctx context.Context, request *SceneListRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Scene

	err = db.NewRaw(sceneListQuery, request.ProjectID, request.Limit).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("scenes.count", len(entities)))

	return entities, nil
}
