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

//go:embed pg.beatsInsert.sql
var beatsInsertQuery string

// ErrBeatsInsertFull is returned by [PgBeatsInsert.Exec] when a beat would take a position past 199,
// so the project would hold more than the 200 beats the schema allows.
var ErrBeatsInsertFull = errors.New("project holds the most beats it can")

// BeatsInsertItem is one beat of a [BeatsInsertRequest].
type BeatsInsertItem struct {
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

// BeatsInsertRequest is the input to [PgBeatsInsert.Exec].
type BeatsInsertRequest struct {
	// ProjectID is the project the beats join.
	ProjectID uuid.UUID

	// Position is the position of the first beat, and the others follow it in order. A position past the
	// last beat appends.
	Position int
	// Beats lists the beats to create, in narrative order.
	Beats []BeatsInsertItem

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgBeatsInsert creates beats in one project from Position on, and moves the beats at or after
// Position down to make room. It returns the new beats in no particular order.
type PgBeatsInsert struct{}

func NewPgBeatsInsert() *PgBeatsInsert {
	return &PgBeatsInsert{}
}

func (dao *PgBeatsInsert) Exec(ctx context.Context, request *BeatsInsertRequest) ([]*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgBeatsInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	ids := lo.Map(request.Beats, func(item BeatsInsertItem, _ int) uuid.UUID { return item.ID })
	titles := lo.Map(request.Beats, func(item BeatsInsertItem, _ int) string { return item.Title })
	colors := lo.Map(request.Beats, func(item BeatsInsertItem, _ int) string { return item.Color })
	descriptions := lo.Map(request.Beats, func(item BeatsInsertItem, _ int) string { return item.Description })
	prompts := lo.Map(request.Beats, func(item BeatsInsertItem, _ int) string { return item.Prompt })

	var entities []*Beat

	err = db.NewRaw(
		beatsInsertQuery,
		request.ProjectID, request.Position, bun.List(ids), bun.List(titles), bun.List(colors),
		bun.List(descriptions), bun.List(prompts), request.Now,
	).Scan(ctx, &entities)
	if isViolation(err, pgCheckViolation, "beats_position_check") {
		err = errors.Join(err, ErrBeatsInsertFull)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("beats.count", len(entities)))

	return entities, nil
}
