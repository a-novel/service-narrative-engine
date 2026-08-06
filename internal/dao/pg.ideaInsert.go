package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.ideaInsert.sql
var ideaInsertQuery string

// IdeaInsertRequest creates a Project and its initial validated Idea Version.
type IdeaInsertRequest struct {
	// ProjectID identifies the stable Project.
	ProjectID uuid.UUID
	// VersionID identifies the initial Idea Version.
	VersionID uuid.UUID
	// OwnerID identifies the user who owns the Project.
	OwnerID uuid.UUID
	// Seed is the source premise supplied by the writer.
	Seed string
	// Genre selects the platform genre vocabulary.
	Genre string
	// Title is the writer-supplied title.
	Title string
	// Now is the logical creation time for both rows.
	Now time.Time
}

// PgIdeaInsert creates a Project with its initial Idea Version.
type PgIdeaInsert struct{}

// NewPgIdeaInsert creates an initial Idea insert operation.
func NewPgIdeaInsert() *PgIdeaInsert {
	return &PgIdeaInsert{}
}

// Exec inserts the Project and Idea Version and returns their combined view.
func (operation *PgIdeaInsert) Exec(ctx context.Context, request *IdeaInsertRequest) (*Idea, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgIdeaInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("idea.version_id", request.VersionID.String()),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var idea Idea

	err = db.NewRaw(
		ideaInsertQuery,
		request.ProjectID,
		request.VersionID,
		request.OwnerID,
		request.Seed,
		request.Genre,
		request.Title,
		request.Now,
	).Scan(ctx, &idea)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &idea), nil
}
