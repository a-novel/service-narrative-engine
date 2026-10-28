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
	// ProjectID is the project whose scenes to read.
	ProjectID uuid.UUID

	// Limit caps the scenes in the page.
	Limit int
	// Offset is how many scenes in narrative order come before the page.
	Offset int
}

// SceneListRow is one scene of a page.
type SceneListRow struct {
	Scene

	// Total counts every scene of the project, beyond the page.
	Total int `bun:"total"`
}

// PgSceneList reads a page of a project's scenes, in narrative order. Each row carries the total,
// and a page past the end comes back empty.
type PgSceneList struct{}

func NewPgSceneList() *PgSceneList {
	return &PgSceneList{}
}

func (dao *PgSceneList) Exec(ctx context.Context, request *SceneListRequest) ([]*SceneListRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*SceneListRow

	err = db.NewRaw(sceneListQuery, request.ProjectID, request.Limit, request.Offset).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("scenes.count", len(entities)))

	return entities, nil
}
