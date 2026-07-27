package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/uptrace/bun/driver/pgdriver"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.generationCallInsert.sql
var generationCallInsertQuery string

// ErrGenerationCallInsertAttemptExists is returned when the job attempt is already recorded.
var ErrGenerationCallInsertAttemptExists = errors.New("generation call attempt already exists")

// ErrGenerationCallInsertJobAlreadySucceeded is returned when another attempt completed the job.
var ErrGenerationCallInsertJobAlreadySucceeded = errors.New("job already has a successful generation call")

// ErrGenerationCallInsertIdeaNotFound is returned when the Idea does not belong to the owner.
var ErrGenerationCallInsertIdeaNotFound = errors.New("idea not found")

// ErrGenerationCallInsertEngineVersionNotFound is returned when the Engine Version does not exist.
var ErrGenerationCallInsertEngineVersionNotFound = errors.New("engine version not found")

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
		attribute.String("generation.id", call.ID.String()),
		attribute.String("generation.job_id", call.JobID.String()),
		attribute.Int("generation.attempt", call.Attempt),
		attribute.String("generation.owner_id", call.OwnerID.String()),
		attribute.String("generation.idea_id", call.IdeaID.String()),
		attribute.String("engine_version.id", call.EngineVersionID.String()),
		attribute.String("generation.provider", call.Provider),
		attribute.String("generation.model", call.Model),
		attribute.String("generation.outcome", string(call.Outcome)),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var generationCall GenerationCall

	err = db.NewRaw(
		generationCallInsertQuery,
		call.ID,
		call.JobID,
		call.Attempt,
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
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) {
			switch pgErr.Field('n') {
			case "generation_calls_job_attempt_key":
				err = errors.Join(err, ErrGenerationCallInsertAttemptExists)
			case "generation_calls_job_success_idx":
				err = errors.Join(err, ErrGenerationCallInsertJobAlreadySucceeded)
			case "generation_calls_idea_owner_fk":
				err = errors.Join(err, ErrGenerationCallInsertIdeaNotFound)
			case "generation_calls_engine_version_fk":
				err = errors.Join(err, ErrGenerationCallInsertEngineVersionNotFound)
			}
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &generationCall), nil
}
