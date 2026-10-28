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

//go:embed pg.sceneSelect.sql
var sceneSelectQuery string

// ErrSceneSelectNotFound is returned by [PgSceneSelect.Exec] when the project holds no scene with the ID.
var ErrSceneSelectNotFound = errors.New("scene not found")

// SceneSelectRequest is the input to [PgSceneSelect.Exec].
type SceneSelectRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID
}

// PgSceneSelect reads a scene of a project by its ID.
type PgSceneSelect struct{}

func NewPgSceneSelect() *PgSceneSelect {
	return &PgSceneSelect{}
}

func (dao *PgSceneSelect) Exec(ctx context.Context, request *SceneSelectRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Scene{}

	err = db.NewRaw(sceneSelectQuery, request.ProjectID, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrSceneSelectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
