package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.beatRecapUpdate.sql
var beatRecapUpdateQuery string

var (
	// ErrBeatRecapUpdateNotFound is returned by [PgBeatRecapUpdate.Exec] when the project holds no beat
	// with the ID.
	ErrBeatRecapUpdateNotFound = errors.New("beat not found")
	// ErrBeatRecapUpdateConflict is returned by [PgBeatRecapUpdate.Exec] when the recap read a version of the
	// beat older than the stored recap's, or newer than the beat's own.
	ErrBeatRecapUpdateConflict = errors.New("recap stamp refused")
)

// BeatRecapUpdateRequest is the input to [PgBeatRecapUpdate.Exec].
type BeatRecapUpdateRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID

	// Recap is a JSON document that follows the consolidate output schema.
	Recap json.RawMessage
	// BeatUpdatedAt is the beat's UpdatedAt the recap read.
	BeatUpdatedAt time.Time
	// Sources is a JSON array of the [BeatRecapSource] the recap read, one per scene of the beat in order.
	Sources json.RawMessage
}

// PgBeatRecapUpdate writes the recap of a beat of the project, leaving the beat's UpdatedAt as it is.
type PgBeatRecapUpdate struct{}

func NewPgBeatRecapUpdate() *PgBeatRecapUpdate {
	return &PgBeatRecapUpdate{}
}

func (dao *PgBeatRecapUpdate) Exec(ctx context.Context, request *BeatRecapUpdateRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgBeatRecapUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var written bool

	err = db.NewRaw(
		beatRecapUpdateQuery,
		request.ProjectID, request.ID, string(request.Recap), request.BeatUpdatedAt, string(request.Sources),
	).Scan(ctx, &written)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		err = errors.Join(err, ErrBeatRecapUpdateNotFound)
	case err == nil && !written:
		err = ErrBeatRecapUpdateConflict
	}

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
