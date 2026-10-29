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

//go:embed pg.beatsUpdate.sql
var beatsUpdateQuery string

// BeatsUpdateItem is one beat of a [BeatsUpdateRequest].
type BeatsUpdateItem struct {
	ID uuid.UUID

	// Title names the beat in the composer.
	Title string
	// Color is the beat's color in the composer, as a lowercase #rrggbb hex code.
	Color string
	// Description says what the beat is for, written for the author.
	Description string
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string
}

// BeatsUpdateRequest is the input to [PgBeatsUpdate.Exec].
type BeatsUpdateRequest struct {
	ProjectID uuid.UUID
	// Beats lists the beats to update, each once.
	Beats []BeatsUpdateItem

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgBeatsUpdate replaces the own fields of beats of a project, and returns them in no particular order. It
// updates all of them, or none when the project does not hold one of them.
type PgBeatsUpdate struct{}

func NewPgBeatsUpdate() *PgBeatsUpdate {
	return &PgBeatsUpdate{}
}

func (dao *PgBeatsUpdate) Exec(ctx context.Context, request *BeatsUpdateRequest) ([]*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgBeatsUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	ids := lo.Map(request.Beats, func(item BeatsUpdateItem, _ int) uuid.UUID { return item.ID })
	titles := lo.Map(request.Beats, func(item BeatsUpdateItem, _ int) string { return item.Title })
	colors := lo.Map(request.Beats, func(item BeatsUpdateItem, _ int) string { return item.Color })
	descriptions := lo.Map(request.Beats, func(item BeatsUpdateItem, _ int) string { return item.Description })
	prompts := lo.Map(request.Beats, func(item BeatsUpdateItem, _ int) string { return item.Prompt })

	var entities []*Beat

	err = db.NewRaw(
		beatsUpdateQuery, request.ProjectID, bun.List(ids), bun.List(titles), bun.List(colors),
		bun.List(descriptions), bun.List(prompts), request.Now,
	).
		Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("beats.count", len(entities)))

	return entities, nil
}
