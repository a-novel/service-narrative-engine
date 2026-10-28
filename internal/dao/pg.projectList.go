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
	// Offset is how many projects newest first come before the page.
	Offset int
}

// ProjectListRow is one project of a page.
type ProjectListRow struct {
	Project

	// Total counts every project of the owner, beyond the page.
	Total int `bun:"total"`
}

// PgProjectList reads a page of the projects an owner holds, newest first. Each row carries the
// total, and a page past the end comes back empty.
type PgProjectList struct{}

func NewPgProjectList() *PgProjectList {
	return &PgProjectList{}
}

func (dao *PgProjectList) Exec(ctx context.Context, request *ProjectListRequest) ([]*ProjectListRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*ProjectListRow

	err = db.NewRaw(projectListQuery, request.OwnerID, request.Limit, request.Offset).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("projects.count", len(entities)))

	return entities, nil
}
