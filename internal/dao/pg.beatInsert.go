package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.beatInsert.sql
var beatInsertQuery string

// ErrBeatInsertFull is returned by [PgBeatInsert.Exec] when the beat would take a position past 199,
// so the project would hold more than the 200 beats the schema allows.
var ErrBeatInsertFull = errors.New("project holds the most beats it can")

// BeatInsertRequest is the input to [PgBeatInsert.Exec].
type BeatInsertRequest struct {
	ID uuid.UUID
	// ProjectID is the project the beat joins.
	ProjectID uuid.UUID

	// Position is the beat's place in narrative order, from 0. A position past the last beat appends.
	Position int

	// Title names the beat in the composer.
	Title string
	// Color is the beat's color in the composer, as a lowercase #rrggbb hex code.
	Color string
	// Description says what the beat is for, written for the author.
	Description string
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgBeatInsert creates a beat at Position, and moves the beats at or after Position down to make
// room.
type PgBeatInsert struct{}

func NewPgBeatInsert() *PgBeatInsert {
	return &PgBeatInsert{}
}

func (dao *PgBeatInsert) Exec(ctx context.Context, request *BeatInsertRequest) (*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgBeatInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Beat{}

	err = db.NewRaw(
		beatInsertQuery,
		request.ID, request.ProjectID, request.Position, request.Title, request.Color,
		request.Description, request.Prompt, request.Now,
	).Scan(ctx, entity)
	if isViolation(err, pgCheckViolation, "beats_position_check") {
		err = errors.Join(err, ErrBeatInsertFull)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
