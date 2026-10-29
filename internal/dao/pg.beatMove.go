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

//go:embed pg.beatMove.sql
var beatMoveQuery string

// ErrBeatMoveNotFound is returned by [PgBeatMove.Exec] when the project holds no beat with the ID.
var ErrBeatMoveNotFound = errors.New("beat not found")

// BeatMoveRequest is the input to [PgBeatMove.Exec].
type BeatMoveRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID

	// Position is the beat's position after the move. A position past the last beat moves it last.
	Position int
}

// PgBeatMove moves a beat to another position, and shifts the beats between its old and new
// position by one toward the old one. The caller locks the project first.
type PgBeatMove struct{}

func NewPgBeatMove() *PgBeatMove {
	return &PgBeatMove{}
}

func (dao *PgBeatMove) Exec(ctx context.Context, request *BeatMoveRequest) (*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgBeatMove")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Beat{}

	err = db.NewRaw(beatMoveQuery, request.ProjectID, request.ID, request.Position).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrBeatMoveNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
