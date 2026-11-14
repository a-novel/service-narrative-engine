package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectLogoDelete.sql
var projectLogoDeleteQuery string

// ProjectLogoDeleteRequest is the input to [PgProjectLogoDelete.Exec].
type ProjectLogoDeleteRequest struct {
	ProjectID uuid.UUID
}

// PgProjectLogoDelete removes the logo of a project in one statement. It returns the project. A project
// without a logo stays as it is, and a missing project fails the write.
type PgProjectLogoDelete struct{}

func NewPgProjectLogoDelete() *PgProjectLogoDelete {
	return &PgProjectLogoDelete{}
}

func (dao *PgProjectLogoDelete) Exec(ctx context.Context, request *ProjectLogoDeleteRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectLogoDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Project{}

	err = db.NewRaw(projectLogoDeleteQuery, request.ProjectID).Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
