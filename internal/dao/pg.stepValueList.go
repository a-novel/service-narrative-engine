package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.stepValueList.sql
var stepValueListQuery string

// StepValueListRequest identifies one opaque key within a Project.
type StepValueListRequest struct {
	ProjectID uuid.UUID
	Key       string
}

// PgStepValueList retrieves the retained history for one Project key.
type PgStepValueList struct{}

// NewPgStepValueList creates a Step Value history operation.
func NewPgStepValueList() *PgStepValueList {
	return &PgStepValueList{}
}

// Exec returns the newest retained Step Values first.
func (operation *PgStepValueList) Exec(
	ctx context.Context,
	request *StepValueListRequest,
) ([]*StepValue, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgStepValueList")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("step_value.key", request.Key),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	values := make([]*StepValue, 0, contentVersionLimit)

	err = db.NewRaw(stepValueListQuery, request.ProjectID, request.Key, contentVersionLimit).
		Scan(ctx, &values)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, values), nil
}
