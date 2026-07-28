package dao

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.stepValueInsert.sql
var stepValueInsertQuery string

// ErrStepValueInsertConflict is returned when content already exists for the
// same Idea, Engine Version, and step.
var ErrStepValueInsertConflict = errors.New("step value already exists")

// StepValueInsertRequest carries validated step content into [PgStepValueInsert.Exec].
type StepValueInsertRequest struct {
	// ID identifies the saved value.
	ID uuid.UUID
	// IdeaID identifies the Idea whose content is being saved.
	IdeaID uuid.UUID
	// EngineVersionID identifies the immutable definition that owns the step.
	EngineVersionID uuid.UUID
	// StepKey identifies the step inside the Engine Version definition.
	StepKey string
	// Value is the schema-validated, source-agnostic content.
	Value json.RawMessage
	// Now is the logical creation time.
	Now time.Time
}

// PgStepValueInsert persists content for one Engine Version step.
type PgStepValueInsert struct{}

// NewPgStepValueInsert creates a step-value insert operation.
func NewPgStepValueInsert() *PgStepValueInsert {
	return &PgStepValueInsert{}
}

// Exec inserts a step value and returns the stored row.
func (operation *PgStepValueInsert) Exec(
	ctx context.Context,
	request *StepValueInsertRequest,
) (*StepValue, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgStepValueInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("step_value.id", request.ID.String()),
		attribute.String("step_value.idea_id", request.IdeaID.String()),
		attribute.String("step_value.engine_version_id", request.EngineVersionID.String()),
		attribute.String("step_value.step_key", request.StepKey),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var stepValue StepValue

	err = db.NewRaw(
		stepValueInsertQuery,
		request.ID,
		request.IdeaID,
		request.EngineVersionID,
		request.StepKey,
		request.Value,
		request.Now,
	).Scan(ctx, &stepValue)
	if err != nil {
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
			err = errors.Join(err, ErrStepValueInsertConflict)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &stepValue), nil
}
