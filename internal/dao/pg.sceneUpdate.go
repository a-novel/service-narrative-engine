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

// ErrSceneUpdateNotFound is returned by [PgSceneUpdate.Exec] when no scene with the ID belongs to a
// project of the owner at the revision.
var ErrSceneUpdateNotFound = errors.New("scene not found at this revision")

// SceneUpdateRequest is the input to [PgSceneUpdate.Exec].
type SceneUpdateRequest struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	// Revision is the revision the caller read.
	Revision int
	Title    string
	Synopsis string
	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgSceneUpdate replaces a scene's own fields when the stored revision is the one the caller read.
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
		sceneUpdateQuery, request.ID, request.OwnerID, request.Revision, request.Title, request.Synopsis, request.Now,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrSceneUpdateNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
