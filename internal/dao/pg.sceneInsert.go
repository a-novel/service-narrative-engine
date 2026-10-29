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

//go:embed pg.sceneInsert.sql
var sceneInsertQuery string

var (
	// ErrSceneInsertFull is returned by [PgSceneInsert.Exec] when the scene would take a position past 9,
	// so the beat would hold more than the 10 scenes the schema allows.
	ErrSceneInsertFull = errors.New("beat holds the most scenes it can")
	// ErrSceneInsertBeatNotFound is returned by [PgSceneInsert.Exec] when the project holds no beat with
	// the ID.
	ErrSceneInsertBeatNotFound = errors.New("beat not found")
)

// SceneInsertRequest is the input to [PgSceneInsert.Exec].
type SceneInsertRequest struct {
	ID uuid.UUID
	// ProjectID is the project the scene joins.
	ProjectID uuid.UUID
	// BeatID is the beat the scene joins.
	BeatID uuid.UUID

	// Position is the scene's place in its beat, from 0. A position past the beat's last scene appends.
	Position int

	// Title names the scene in the composer.
	Title string
	// Description refines the beat's description for the scene.
	Description string
	// Prompt refines the beat's prompt for the scene.
	Prompt string

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgSceneInsert creates a scene at Position in its beat, and moves the beat's scenes at or after Position
// down to make room.
type PgSceneInsert struct{}

func NewPgSceneInsert() *PgSceneInsert {
	return &PgSceneInsert{}
}

func (dao *PgSceneInsert) Exec(ctx context.Context, request *SceneInsertRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Scene{}

	err = db.NewRaw(
		sceneInsertQuery,
		request.ID, request.ProjectID, request.BeatID, request.Position, request.Title,
		request.Description, request.Prompt, request.Now,
	).Scan(ctx, entity)

	switch {
	case isViolation(err, pgCheckViolation, "scenes_position_check"):
		err = errors.Join(err, ErrSceneInsertFull)
	case isViolation(err, pgForeignKeyViolation, "scenes_beat_fkey"):
		err = errors.Join(err, ErrSceneInsertBeatNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
