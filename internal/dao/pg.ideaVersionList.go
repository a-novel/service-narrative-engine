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

//go:embed pg.ideaVersionList.sql
var ideaVersionListQuery string

// IdeaVersionListRequest identifies the Project whose retained Idea history is requested.
type IdeaVersionListRequest struct {
	ProjectID uuid.UUID
}

// PgIdeaVersionList retrieves the retained Idea Versions for one Project.
type PgIdeaVersionList struct{}

// NewPgIdeaVersionList creates an Idea history operation.
func NewPgIdeaVersionList() *PgIdeaVersionList {
	return &PgIdeaVersionList{}
}

// Exec returns the newest retained Idea Versions first.
func (operation *PgIdeaVersionList) Exec(
	ctx context.Context,
	request *IdeaVersionListRequest,
) ([]*IdeaVersion, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgIdeaVersionList")
	defer span.End()

	span.SetAttributes(attribute.String("project.id", request.ProjectID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	versions := make([]*IdeaVersion, 0, contentVersionLimit)

	err = db.NewRaw(ideaVersionListQuery, request.ProjectID, contentVersionLimit).
		Scan(ctx, &versions)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, versions), nil
}
