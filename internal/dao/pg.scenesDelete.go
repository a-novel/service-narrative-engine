package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.scenesDelete.sql
var scenesDeleteQuery string

// ScenesDeleteRequest is the input to [PgScenesDelete.Exec].
type ScenesDeleteRequest struct {
	ProjectID uuid.UUID
	// IDs lists the scenes to delete.
	IDs []uuid.UUID
}

// PgScenesDelete removes scenes of a project with their cells, and moves the scenes after them up to close
// the gaps. It returns the deleted scenes, in no particular order, and skips an ID the project does not
// hold.
type PgScenesDelete struct{}

func NewPgScenesDelete() *PgScenesDelete {
	return &PgScenesDelete{}
}

func (dao *PgScenesDelete) Exec(ctx context.Context, request *ScenesDeleteRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgScenesDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Scene

	err = db.NewRaw(scenesDeleteQuery, request.ProjectID, bun.List(request.IDs)).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("scenes.count", len(entities)))

	return entities, nil
}
