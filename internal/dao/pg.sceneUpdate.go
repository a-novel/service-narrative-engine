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

//go:embed pg.sceneUpdate.sql
var sceneUpdateQuery string

// ErrSceneUpdateNotFound is returned by [PgSceneUpdate.Exec] when the project holds no scene with the ID.
var ErrSceneUpdateNotFound = errors.New("scene not found")

// SceneUpdateRequest is the input to [PgSceneUpdate.Exec].
type SceneUpdateRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID

	// Title names the scene in the composer.
	Title string
	// Description refines the beat's description for the scene.
	Description string
	// Prompt refines the beat's prompt for the scene.
	Prompt string

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgSceneUpdate replaces a scene's own fields.
type PgSceneUpdate struct{}

func NewPgSceneUpdate() *PgSceneUpdate {
	return &PgSceneUpdate{}
}

func (dao *PgSceneUpdate) Exec(ctx context.Context, request *SceneUpdateRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Scene{}

	err = db.NewRaw(
		sceneUpdateQuery, request.ProjectID, request.ID,
		request.Title, request.Description, request.Prompt, request.Now,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrSceneUpdateNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
