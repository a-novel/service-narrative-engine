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

//go:embed pg.scenesInsert.sql
var scenesInsertQuery string

// ErrScenesInsertFull is returned by [PgScenesInsert.Exec] when a scene would take a position past 499,
// so the project would hold more than the 500 scenes the schema allows.
var ErrScenesInsertFull = errors.New("project holds the most scenes it can")

// ScenesInsertItem is one scene of a [ScenesInsertRequest].
type ScenesInsertItem struct {
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

// ScenesInsertRequest is the input to [PgScenesInsert.Exec].
type ScenesInsertRequest struct {
	// ProjectID is the project the scenes join.
	ProjectID uuid.UUID

	// Position is the position of the first scene, and the others follow it in order. A position past the
	// last scene appends.
	Position int
	// Scenes lists the scenes to create, in narrative order.
	Scenes []ScenesInsertItem

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgScenesInsert creates scenes in one project from Position on, and moves the scenes at or after
// Position down to make room. It returns the new scenes in no particular order.
type PgScenesInsert struct{}

func NewPgScenesInsert() *PgScenesInsert {
	return &PgScenesInsert{}
}

func (dao *PgScenesInsert) Exec(ctx context.Context, request *ScenesInsertRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgScenesInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	ids := lo.Map(request.Scenes, func(item ScenesInsertItem, _ int) uuid.UUID { return item.ID })
	titles := lo.Map(request.Scenes, func(item ScenesInsertItem, _ int) string { return item.Title })
	colors := lo.Map(request.Scenes, func(item ScenesInsertItem, _ int) string { return item.Color })
	descriptions := lo.Map(request.Scenes, func(item ScenesInsertItem, _ int) string { return item.Description })
	prompts := lo.Map(request.Scenes, func(item ScenesInsertItem, _ int) string { return item.Prompt })

	var entities []*Scene

	err = db.NewRaw(
		scenesInsertQuery,
		request.ProjectID, request.Position, bun.List(ids), bun.List(titles), bun.List(colors),
		bun.List(descriptions), bun.List(prompts), request.Now,
	).Scan(ctx, &entities)
	if isViolation(err, pgCheckViolation, "scenes_position_check") {
		err = errors.Join(err, ErrScenesInsertFull)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("scenes.count", len(entities)))

	return entities, nil
}
