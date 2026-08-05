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

//go:embed pg.stepValueInsert.sql
var stepValueInsertQuery string

//go:embed pg.stepValuePrune.sql
var stepValuePruneQuery string

// StepValueInsertRequest carries bounded JSON into [PgStepValueInsert.Exec].
type StepValueInsertRequest struct {
	// ID identifies the saved version.
	ID uuid.UUID
	// ProjectID identifies the Project whose content is being saved.
	ProjectID uuid.UUID
	// OwnerID identifies the user who owns the Project.
	OwnerID uuid.UUID
	// Key is an opaque client-controlled content identity.
	Key string
	// Value is arbitrary valid JSON.
	Value json.RawMessage
	// Now is the logical creation time.
	Now time.Time
}

// PgStepValueInsert persists one opaque Project value version.
type PgStepValueInsert struct{}

// NewPgStepValueInsert creates a Step Value insert operation.
func NewPgStepValueInsert() *PgStepValueInsert {
	return &PgStepValueInsert{}
}

// Exec appends a Step Value and retains the newest versions for its Project and key.
func (operation *PgStepValueInsert) Exec(
	ctx context.Context,
	request *StepValueInsertRequest,
) (*StepValue, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgStepValueInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("step_value.id", request.ID.String()),
		attribute.String("step_value.project_id", request.ProjectID.String()),
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

	var stepValue StepValue

	err = db.NewRaw(
		stepValueInsertQuery,
		request.ID,
		request.ProjectID,
		request.Key,
		request.Value,
		request.Now,
	).Scan(ctx, &stepValue)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute insert query: %w", err))
	}

	_, err = db.NewRaw(
		stepValuePruneQuery,
		request.ProjectID,
		request.Key,
		contentVersionLimit,
	).Exec(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute prune query: %w", err))
	}

	return otel.ReportSuccess(span, &stepValue), nil
}
