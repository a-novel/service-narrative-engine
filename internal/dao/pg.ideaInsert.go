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

// IdeaInsertRequest carries a validated Idea into [PgIdeaInsert.Exec].
type IdeaInsertRequest struct {
	// ID identifies the Idea.
	ID uuid.UUID
	// OwnerID identifies the user who owns the Idea.
	OwnerID uuid.UUID
	// Seed is the source premise supplied by the writer.
	Seed string
	// StoryType selects the platform story shape.
	StoryType string
	// Genre selects the platform genre vocabulary.
	Genre string
	// Title is the optional title supplied by the writer.
	Title *string
	// Now is the logical creation time.
	Now time.Time
}

// PgIdeaInsert persists a typed Idea.
type PgIdeaInsert struct{}

// NewPgIdeaInsert creates an Idea insert operation.
func NewPgIdeaInsert() *PgIdeaInsert {
	return &PgIdeaInsert{}
}

// Exec inserts an Idea and returns the stored row.
func (operation *PgIdeaInsert) Exec(ctx context.Context, request *IdeaInsertRequest) (*Idea, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgIdeaInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("idea.id", request.ID.String()),
		attribute.String("idea.owner_id", request.OwnerID.String()),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var idea Idea

	err = db.NewRaw(
		ideaInsertQuery,
		request.ID,
		request.OwnerID,
		request.Seed,
		request.StoryType,
		request.Genre,
		request.Title,
		request.Now,
	).Scan(ctx, &idea)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &idea), nil
}
