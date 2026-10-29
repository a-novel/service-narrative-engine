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

//go:embed pg.sceneDelete.sql
var sceneDeleteQuery string

// ErrSceneDeleteNotFound is returned by [PgSceneDelete.Exec] when the project holds no scene with the ID.
var ErrSceneDeleteNotFound = errors.New("scene not found")

// SceneDeleteRequest is the input to [PgSceneDelete.Exec].
type SceneDeleteRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID
}

// PgSceneDelete removes a scene with its cells, and moves the scenes after it in its beat up by one.
type PgSceneDelete struct{}

func NewPgSceneDelete() *PgSceneDelete {
	return &PgSceneDelete{}
}

func (dao *PgSceneDelete) Exec(ctx context.Context, request *SceneDeleteRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Scene{}

	err = db.NewRaw(sceneDeleteQuery, request.ProjectID, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrSceneDeleteNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
