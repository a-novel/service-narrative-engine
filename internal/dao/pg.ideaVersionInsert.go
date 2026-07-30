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

//go:embed pg.ideaVersionInsert.sql
var ideaVersionInsertQuery string

//go:embed pg.ideaVersionPrune.sql
var ideaVersionPruneQuery string

// IdeaVersionInsertRequest carries validated Idea content into [PgIdeaVersionInsert.Exec].
type IdeaVersionInsertRequest struct {
	// ID identifies the content version.
	ID uuid.UUID
	// IdeaID identifies the stable Idea root.
	IdeaID uuid.UUID
	// OwnerID identifies the user who owns the Idea.
	OwnerID uuid.UUID
	// Seed is the source premise supplied by the writer.
	Seed string
	// Genre selects the platform genre vocabulary.
	Genre string
	// Title is the writer-supplied title, or an empty string when omitted.
	Title string
	// Now is the logical save time.
	Now time.Time
}

// PgIdeaVersionInsert saves one Idea content version and bounds its history.
type PgIdeaVersionInsert struct{}

// NewPgIdeaVersionInsert creates an Idea-version insert operation.
func NewPgIdeaVersionInsert() *PgIdeaVersionInsert {
	return &PgIdeaVersionInsert{}
}

// Exec inserts one Idea content version and removes versions beyond the retention limit.
func (operation *PgIdeaVersionInsert) Exec(
	ctx context.Context,
	request *IdeaVersionInsertRequest,
) (*IdeaVersion, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgIdeaVersionInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("idea.id", request.IdeaID.String()),
		attribute.String("idea.version_id", request.ID.String()),
		attribute.String("idea.owner_id", request.OwnerID.String()),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	err = requireVersionedWriteTransaction(ctx)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = lockIdea(ctx, db, request.IdeaID, request.OwnerID)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("lock Idea: %w", err))
	}

	var ideaVersion IdeaVersion

	err = db.NewRaw(
		ideaVersionInsertQuery,
		request.ID,
		request.IdeaID,
		request.Seed,
		request.Genre,
		request.Title,
		request.Now,
	).Scan(ctx, &ideaVersion)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute insert query: %w", err))
	}

	_, err = db.NewRaw(ideaVersionPruneQuery, request.IdeaID, contentVersionLimit).Exec(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute prune query: %w", err))
	}

	return otel.ReportSuccess(span, &ideaVersion), nil
}
