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

//go:embed pg.sceneInsert.sql
var sceneInsertQuery string

// SceneInsertItem is one scene of a [SceneInsertRequest].
type SceneInsertItem struct {
	ID          uuid.UUID
	Title       string
	Color       string
	Description string
	Prompt      string
}

// SceneInsertRequest is the input to [PgSceneInsert.Exec].
type SceneInsertRequest struct {
	ProjectID uuid.UUID
	// Position is the position of the first scene. The others follow it in order.
	Position int
	Scenes   []SceneInsertItem
	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgSceneInsert creates scenes in one project, in one statement. It returns them in no particular
// order.
type PgSceneInsert struct{}

func NewPgSceneInsert() *PgSceneInsert {
	return &PgSceneInsert{}
}

func (dao *PgSceneInsert) Exec(ctx context.Context, request *SceneInsertRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	ids := lo.Map(request.Scenes, func(item SceneInsertItem, _ int) uuid.UUID { return item.ID })
	titles := lo.Map(request.Scenes, func(item SceneInsertItem, _ int) string { return item.Title })
	colors := lo.Map(request.Scenes, func(item SceneInsertItem, _ int) string { return item.Color })
	descriptions := lo.Map(request.Scenes, func(item SceneInsertItem, _ int) string { return item.Description })
	prompts := lo.Map(request.Scenes, func(item SceneInsertItem, _ int) string { return item.Prompt })

	var entities []*Scene

	err = db.NewRaw(
		sceneInsertQuery,
		request.ProjectID, request.Position, bun.List(ids), bun.List(titles), bun.List(colors),
		bun.List(descriptions), bun.List(prompts), request.Now,
	).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("scenes.count", len(entities)))

	return entities, nil
}
