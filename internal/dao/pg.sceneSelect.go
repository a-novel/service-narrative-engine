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

// ErrSceneSelectNotFound is returned by [PgSceneSelect.Exec] when no scene with the ID belongs to a
// project of the owner.
var ErrSceneSelectNotFound = errors.New("scene not found")

// SceneSelectRequest is the input to [PgSceneSelect.Exec].
type SceneSelectRequest struct {
	ID uuid.UUID
	// OwnerID is the user who must own the scene's project. Another owner's scene reads as not found.
	OwnerID uuid.UUID
}

// PgSceneSelect reads a scene of a project its owner holds.
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

	err = db.NewRaw(sceneSelectQuery, request.ID, request.OwnerID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrSceneSelectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
