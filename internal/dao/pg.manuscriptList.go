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

//go:embed pg.manuscriptList.sql
var manuscriptListQuery string

// ManuscriptListRequest identifies the Project whose retained Manuscript history is requested.
type ManuscriptListRequest struct {
	ProjectID uuid.UUID
}

// PgManuscriptList retrieves the retained Manuscript Versions for one Project.
type PgManuscriptList struct{}

// NewPgManuscriptList creates a Manuscript history operation.
func NewPgManuscriptList() *PgManuscriptList {
	return &PgManuscriptList{}
}

// Exec returns the newest retained Manuscript Versions first.
func (operation *PgManuscriptList) Exec(
	ctx context.Context,
	request *ManuscriptListRequest,
) ([]*Manuscript, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgManuscriptList")
	defer span.End()

	span.SetAttributes(attribute.String("project.id", request.ProjectID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	versions := make([]*Manuscript, 0, contentVersionLimit)

	err = db.NewRaw(manuscriptListQuery, request.ProjectID, contentVersionLimit).
		Scan(ctx, &versions)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, versions), nil
}
