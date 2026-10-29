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

//go:embed pg.scenesUpdate.sql
var scenesUpdateQuery string

// ScenesUpdateItem is one scene of a [ScenesUpdateRequest].
type ScenesUpdateItem struct {
	ID uuid.UUID

	// Title names the scene in the composer.
	Title string
	// Description refines the beat's description for the scene.
	Description string
	// Prompt refines the beat's prompt for the scene.
	Prompt string
}

// ScenesUpdateRequest is the input to [PgScenesUpdate.Exec].
type ScenesUpdateRequest struct {
	ProjectID uuid.UUID
	// Scenes lists the scenes to update, each once.
	Scenes []ScenesUpdateItem

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgScenesUpdate replaces the own fields of scenes of a project, and returns them in no particular order. It
// updates all of them, or none when the project does not hold one of them.
type PgScenesUpdate struct{}

func NewPgScenesUpdate() *PgScenesUpdate {
	return &PgScenesUpdate{}
}

func (dao *PgScenesUpdate) Exec(ctx context.Context, request *ScenesUpdateRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgScenesUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	ids := lo.Map(request.Scenes, func(item ScenesUpdateItem, _ int) uuid.UUID { return item.ID })
	titles := lo.Map(request.Scenes, func(item ScenesUpdateItem, _ int) string { return item.Title })
	descriptions := lo.Map(request.Scenes, func(item ScenesUpdateItem, _ int) string { return item.Description })
	prompts := lo.Map(request.Scenes, func(item ScenesUpdateItem, _ int) string { return item.Prompt })

	var entities []*Scene

	err = db.NewRaw(
		scenesUpdateQuery, request.ProjectID, bun.List(ids), bun.List(titles),
		bun.List(descriptions), bun.List(prompts), request.Now,
	).
		Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("scenes.count", len(entities)))

	return entities, nil
}
