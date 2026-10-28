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

// ErrSceneInsertFull is returned by [PgSceneInsert.Exec] when the scene would take a position past 499,
// so the project would hold more than the 500 scenes the schema allows.
var ErrSceneInsertFull = errors.New("project holds the most scenes it can")

// SceneInsertRequest is the input to [PgSceneInsert.Exec].
type SceneInsertRequest struct {
	ID uuid.UUID
	// ProjectID is the project the scene joins.
	ProjectID uuid.UUID

	// Position is the scene's place in narrative order, from 0. A position past the last scene appends.
	Position int

	// Title names the scene in the composer.
	Title string
	// Color is the scene's color in the composer, as a lowercase #rrggbb hex code.
	Color string
	// Description says what the scene is for, written for the author.
	Description string
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgSceneInsert creates a scene at Position, and moves the scenes at or after Position down to make
// room.
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
		request.ID, request.ProjectID, request.Position, request.Title, request.Color,
		request.Description, request.Prompt, request.Now,
	).Scan(ctx, entity)
	if isViolation(err, pgCheckViolation, "scenes_position_check") {
		err = errors.Join(err, ErrSceneInsertFull)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
