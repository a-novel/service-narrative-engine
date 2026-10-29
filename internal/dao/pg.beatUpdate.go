package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.beatUpdate.sql
var beatUpdateQuery string

// ErrBeatUpdateNotFound is returned by [PgBeatUpdate.Exec] when the project holds no beat with the ID.
var ErrBeatUpdateNotFound = errors.New("beat not found")

// BeatUpdateRequest is the input to [PgBeatUpdate.Exec].
type BeatUpdateRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID

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

// PgBeatUpdate replaces a beat's own fields.
type PgBeatUpdate struct{}

func NewPgBeatUpdate() *PgBeatUpdate {
	return &PgBeatUpdate{}
}

func (dao *PgBeatUpdate) Exec(ctx context.Context, request *BeatUpdateRequest) (*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgBeatUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Beat{}

	err = db.NewRaw(
		beatUpdateQuery, request.ProjectID, request.ID,
		request.Title, request.Color, request.Description, request.Prompt, request.Now,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrBeatUpdateNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
