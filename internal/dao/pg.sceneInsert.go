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

//go:embed pg.sceneInsert.sql
var sceneInsertQuery string

// ErrSceneInsertFull is returned by [PgSceneInsert.Exec] when a scene would take a position past 499,
// so the project would hold more than the 500 scenes the schema allows.
var ErrSceneInsertFull = errors.New("project holds the most scenes it can")

// SceneInsertItem is one scene of a [SceneInsertRequest].
type SceneInsertItem struct {
	ID uuid.UUID

	// Title names the scene in the composer.
	Title string
	// Color is the scene's color in the composer, as a lowercase #rrggbb hex code.
	Color string
	// Description says what the scene is for, written for the author.
	Description string
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string
}

// SceneInsertRequest is the input to [PgSceneInsert.Exec].
type SceneInsertRequest struct {
	// ProjectID is the project the scenes join.
	ProjectID uuid.UUID

	// Position is the position of the first scene, and the others follow it in order. A position past the
	// last scene appends.
	Position int
	// Scenes lists the scenes to create, in narrative order.
	Scenes []SceneInsertItem

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgSceneInsert creates scenes in one project from Position on, and moves the scenes at or after
// Position down to make room. It returns the new scenes in no particular order.
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
	if isViolation(err, pgCheckViolation, "scenes_position_check") {
		err = errors.Join(err, ErrSceneInsertFull)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("scenes.count", len(entities)))

	return entities, nil
}
