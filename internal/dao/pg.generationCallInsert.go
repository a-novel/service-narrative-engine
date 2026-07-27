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

// GenerationCallInsertRequest carries provider usage into [PgGenerationCallInsert.Exec].
type GenerationCallInsertRequest struct {
	// Call is the immutable usage row to persist.
	Call *GenerationCall
}

// PgGenerationCallInsert persists the pricing inputs for a provider exchange.
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
		attribute.Int("generation.attempt", call.Attempt),
		attribute.String("generation.owner_id", call.OwnerID.String()),
		attribute.String("generation.provider", call.Provider),
		attribute.String("generation.model", call.Model),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var generationCall GenerationCall

	err = db.NewRaw(
		generationCallInsertQuery,
		call.JobID,
		call.Attempt,
		call.OwnerID,
		call.Provider,
		call.Model,
		call.InputTokens,
		call.OutputTokens,
		call.CreatedAt,
	).Scan(ctx, &generationCall)
	if err != nil {
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) {
			if pgErr.Field('n') == "generation_calls_pkey" {
				err = errors.Join(err, ErrGenerationCallInsertAttemptExists)
			}
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &generationCall), nil
}
