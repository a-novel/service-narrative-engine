package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.generationCallInsert.sql
var generationCallInsertQuery string

// GenerationCallInsertRequest carries a completed provider exchange into [PgGenerationCallInsert.Exec].
type GenerationCallInsertRequest struct {
	// Call is the immutable audit row to persist.
	Call *GenerationCall
}

// PgGenerationCallInsert persists a terminal provider exchange.
type PgGenerationCallInsert struct{}

// NewPgGenerationCallInsert creates a Generation Call insert operation.
func NewPgGenerationCallInsert() *PgGenerationCallInsert {
	return &PgGenerationCallInsert{}
}

// Exec inserts the audit row and returns the stored record.
func (operation *PgGenerationCallInsert) Exec(
	ctx context.Context,
	request *GenerationCallInsertRequest,
) (*GenerationCall, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgGenerationCallInsert")
	defer span.End()

	call := request.Call
	span.SetAttributes(
		attribute.String("generation.job_id", call.JobID.String()),
		attribute.String("generation.owner_id", call.OwnerID.String()),
		attribute.String("generation.idea_id", call.IdeaID.String()),
		attribute.String("engine_version.id", call.EngineVersionID.String()),
		attribute.String("generation.provider", call.Provider),
		attribute.String("generation.model", call.Model),
		attribute.String("generation.outcome", call.Outcome),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var generationCall GenerationCall

	err = db.NewRaw(
		generationCallInsertQuery,
		call.JobID,
		call.OwnerID,
		call.IdeaID,
		call.EngineVersionID,
		call.Provider,
		call.ProviderCallID,
		call.RequestHash,
		call.Model,
		call.Outcome,
		call.RawOutput,
		call.InputTokens,
		call.OutputTokens,
		call.TotalTokens,
		call.LatencyMilliseconds,
		call.Refusal,
		call.Error,
		call.CreatedAt,
		call.CompletedAt,
	).Scan(ctx, &generationCall)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &generationCall), nil
}
