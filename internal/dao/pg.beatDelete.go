package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.beatDelete.sql
var beatDeleteQuery string

// ErrBeatDeleteNotFound is returned by [PgBeatDelete.Exec] when the project holds no beat with the ID.
var ErrBeatDeleteNotFound = errors.New("beat not found")

// BeatDeleteRequest is the input to [PgBeatDelete.Exec].
type BeatDeleteRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID
}

// PgBeatDelete removes a beat with its scenes and their cells, and moves the beats after it up by one.
type PgBeatDelete struct{}

func NewPgBeatDelete() *PgBeatDelete {
	return &PgBeatDelete{}
}

func (dao *PgBeatDelete) Exec(ctx context.Context, request *BeatDeleteRequest) (*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgBeatDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Beat{}

	err = db.NewRaw(beatDeleteQuery, request.ProjectID, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrBeatDeleteNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
