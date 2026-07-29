package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.stepValueSelectLatest.sql
var stepValueSelectLatestQuery string

// StepValueSelectLatestRequest identifies project context and keys replaced by overrides.
type StepValueSelectLatestRequest struct {
	// IdeaID identifies the project whose current step context is requested.
	IdeaID uuid.UUID
	// ExcludeStepKeys prevents overridden keys from being fetched from storage.
	ExcludeStepKeys []string
}

// PgStepValueSelectLatest selects the newest saved value for every logical step key.
type PgStepValueSelectLatest struct{}

// NewPgStepValueSelectLatest creates a latest-step-value select operation.
func NewPgStepValueSelectLatest() *PgStepValueSelectLatest {
	return &PgStepValueSelectLatest{}
}

// Exec returns one current value per key across every Engine Version.
func (operation *PgStepValueSelectLatest) Exec(
	ctx context.Context,
	request *StepValueSelectLatestRequest,
) ([]*StepValue, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgStepValueSelectLatest")
	defer span.End()

	span.SetAttributes(
		attribute.String("step_value.idea_id", request.IdeaID.String()),
		attribute.Int("step_value.excluded_key_count", len(request.ExcludeStepKeys)),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	stepValues := make([]*StepValue, 0)

	err = db.NewRaw(
		stepValueSelectLatestQuery,
		request.IdeaID,
		pgdialect.Array(request.ExcludeStepKeys),
	).Scan(ctx, &stepValues)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, stepValues), nil
}
