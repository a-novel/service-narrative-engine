package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectList.sql
var projectListQuery string

// ProjectListRequest is the input to [PgProjectList.Exec].
type ProjectListRequest struct {
	// OwnerID is the user whose projects to read.
	OwnerID uuid.UUID

	// Limit caps the projects in the page.
	Limit int
	// After is the ID the page starts after, in descending order. uuid.Max starts at the newest project.
	After uuid.UUID
}

// PgProjectList reads a page of the projects an owner holds, newest first.
type PgProjectList struct{}

func NewPgProjectList() *PgProjectList {
	return &PgProjectList{}
}

func (dao *PgProjectList) Exec(ctx context.Context, request *ProjectListRequest) ([]*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Project

	err = db.NewRaw(projectListQuery, request.OwnerID, request.Limit, request.After).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("projects.count", len(entities)))

	return entities, nil
}
