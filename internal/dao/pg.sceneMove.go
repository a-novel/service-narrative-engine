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

//go:embed pg.sceneMove.sql
var sceneMoveQuery string

var (
	// ErrSceneMoveNotFound is returned by [PgSceneMove.Exec] when the project holds no scene with the ID.
	ErrSceneMoveNotFound = errors.New("scene not found")
	// ErrSceneMoveBeatNotFound is returned by [PgSceneMove.Exec] when the project holds no beat with the
	// target ID.
	ErrSceneMoveBeatNotFound = errors.New("beat not found")
	// ErrSceneMoveFull is returned by [PgSceneMove.Exec] when the target beat already holds the 10 scenes
	// the schema allows.
	ErrSceneMoveFull = errors.New("beat holds the most scenes it can")
)

// SceneMoveRequest is the input to [PgSceneMove.Exec].
type SceneMoveRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID

	// BeatID is the beat that holds the scene after the move, its own or another of the project.
	BeatID uuid.UUID
	// Position is the scene's position in that beat after the move. A position past the beat's last scene
	// moves it last.
	Position int
}

// PgSceneMove moves a scene to a position in a beat of its project. Its old beat closes the gap and the
// target beat opens one, so the scene keeps its ID and its cells. The caller locks the project first.
type PgSceneMove struct{}

func NewPgSceneMove() *PgSceneMove {
	return &PgSceneMove{}
}

func (dao *PgSceneMove) Exec(ctx context.Context, request *SceneMoveRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneMove")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Scene{}

	err = db.NewRaw(sceneMoveQuery, request.ProjectID, request.ID, request.BeatID, request.Position).Scan(ctx, entity)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		err = errors.Join(err, ErrSceneMoveNotFound)
	case isViolation(err, pgForeignKeyViolation, "scenes_beat_fkey"):
		err = errors.Join(err, ErrSceneMoveBeatNotFound)
	case isViolation(err, pgCheckViolation, "scenes_position_check"):
		err = errors.Join(err, ErrSceneMoveFull)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
