package dao

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.manuscriptInsert.sql
var manuscriptInsertQuery string

//go:embed pg.manuscriptPrune.sql
var manuscriptPruneQuery string

// ManuscriptInsertRequest carries a validated Manuscript into [PgManuscriptInsert.Exec].
type ManuscriptInsertRequest struct {
	// ID identifies the Manuscript Version.
	ID uuid.UUID
	// ProjectID identifies the Project whose Manuscript is being saved.
	ProjectID uuid.UUID
	// OwnerID identifies the user who owns the Project.
	OwnerID uuid.UUID
	// Value is the self-contained Manuscript document.
	Value json.RawMessage
	// Now is the logical creation time.
	Now time.Time
}

// PgManuscriptInsert persists one Manuscript Version.
type PgManuscriptInsert struct{}

// NewPgManuscriptInsert creates a Manuscript insert operation.
func NewPgManuscriptInsert() *PgManuscriptInsert {
	return &PgManuscriptInsert{}
}

// Exec appends a Manuscript and retains the Project's newest versions.
func (operation *PgManuscriptInsert) Exec(
	ctx context.Context,
	request *ManuscriptInsertRequest,
) (*Manuscript, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgManuscriptInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("manuscript.id", request.ID.String()),
		attribute.String("manuscript.project_id", request.ProjectID.String()),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	err = requireVersionedWriteTransaction(ctx)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = lockProject(ctx, db, request.ProjectID, request.OwnerID)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("lock Project: %w", err))
	}

	var manuscript Manuscript

	err = db.NewRaw(
		manuscriptInsertQuery,
		request.ID,
		request.ProjectID,
		request.Value,
		request.Now,
	).Scan(ctx, &manuscript)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute insert query: %w", err))
	}

	_, err = db.NewRaw(manuscriptPruneQuery, request.ProjectID, contentVersionLimit).Exec(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute prune query: %w", err))
	}

	return otel.ReportSuccess(span, &manuscript), nil
}
