package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sceneLength.sql
var sceneLengthQuery string

// SceneLength is how many characters the cells of one scene hold.
type SceneLength struct {
	SceneID uuid.UUID `bun:"scene_id,type:uuid"`

	// Length counts the characters of the scene's cells, over their stored JSON documents.
	Length int `bun:"length"`
}

// SceneLengthRequest is the input to [PgSceneLength.Exec].
type SceneLengthRequest struct {
	ProjectID uuid.UUID
	// IDs lists the scenes to measure.
	IDs []uuid.UUID
}

// PgSceneLength reads how many characters the cells of scenes of a project hold, in no particular order.
// It skips a scene without cells and an ID the project does not hold.
type PgSceneLength struct{}

func NewPgSceneLength() *PgSceneLength {
	return &PgSceneLength{}
}

func (dao *PgSceneLength) Exec(ctx context.Context, request *SceneLengthRequest) ([]*SceneLength, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneLength")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*SceneLength

	err = db.NewRaw(sceneLengthQuery, request.ProjectID, bun.List(request.IDs), len(request.IDs)).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entities, nil
}
