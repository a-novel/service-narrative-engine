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

// ErrSceneMoveNotFound is returned by [PgSceneMove.Exec] when no scene has the ID.
var ErrSceneMoveNotFound = errors.New("scene not found")

// SceneMoveRequest is the input to [PgSceneMove.Exec].
type SceneMoveRequest struct {
	ID uuid.UUID

	// To is the scene's position after the move.
	To int
}

// PgSceneMove moves a scene to another position, and shifts the scenes between its old and new
// position by one toward the old one. The caller locks the project first.
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

	err = db.NewRaw(sceneMoveQuery, request.ID, request.To).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrSceneMoveNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
