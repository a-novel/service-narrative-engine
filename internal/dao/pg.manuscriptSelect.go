package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.manuscriptSelect.sql
var manuscriptSelectQuery string

// ManuscriptSelectRequest identifies the Project whose current Manuscript is requested.
type ManuscriptSelectRequest struct {
	ProjectID uuid.UUID
}

// PgManuscriptSelect retrieves the latest Manuscript for one Project.
type PgManuscriptSelect struct{}

// NewPgManuscriptSelect creates a current Manuscript operation.
func NewPgManuscriptSelect() *PgManuscriptSelect {
	return &PgManuscriptSelect{}
}

// Exec returns the latest Manuscript, or nil when the Project has none.
func (operation *PgManuscriptSelect) Exec(
	ctx context.Context,
	request *ManuscriptSelectRequest,
) (*Manuscript, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgManuscriptSelect")
	defer span.End()

	span.SetAttributes(attribute.String("project.id", request.ProjectID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var manuscript Manuscript

	err = db.NewRaw(manuscriptSelectQuery, request.ProjectID).Scan(ctx, &manuscript)
	if errors.Is(err, sql.ErrNoRows) {
		return otel.ReportSuccess(span, (*Manuscript)(nil)), nil
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &manuscript), nil
}
