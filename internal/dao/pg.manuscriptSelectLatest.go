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

//go:embed pg.manuscriptSelectLatest.sql
var manuscriptSelectLatestQuery string

// ErrManuscriptSelectLatestNotFound reports a project without a saved Manuscript.
var ErrManuscriptSelectLatestNotFound = errors.New("manuscript not found")

// ManuscriptSelectLatestRequest identifies the project whose current Manuscript is requested.
type ManuscriptSelectLatestRequest struct {
	// IdeaID identifies the project.
	IdeaID uuid.UUID
}

// PgManuscriptSelectLatest selects the project's most recently saved Manuscript.
type PgManuscriptSelectLatest struct{}

// NewPgManuscriptSelectLatest creates a latest-Manuscript select operation.
func NewPgManuscriptSelectLatest() *PgManuscriptSelectLatest {
	return &PgManuscriptSelectLatest{}
}

// Exec returns the current Manuscript for one project.
func (operation *PgManuscriptSelectLatest) Exec(
	ctx context.Context,
	request *ManuscriptSelectLatestRequest,
) (*Manuscript, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgManuscriptSelectLatest")
	defer span.End()

	span.SetAttributes(attribute.String("manuscript.idea_id", request.IdeaID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var manuscript Manuscript

	err = db.NewRaw(manuscriptSelectLatestQuery, request.IdeaID).Scan(ctx, &manuscript)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrManuscriptSelectLatestNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &manuscript), nil
}
