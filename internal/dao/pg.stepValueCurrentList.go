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

//go:embed pg.stepValueCurrentList.sql
var stepValueCurrentListQuery string

// StepValueCurrentListRequest identifies the Project whose current Step Values are requested.
type StepValueCurrentListRequest struct {
	ProjectID uuid.UUID
}

// PgStepValueCurrentList retrieves the latest saved value for every Project key.
type PgStepValueCurrentList struct{}

// NewPgStepValueCurrentList creates a current Step Value operation.
func NewPgStepValueCurrentList() *PgStepValueCurrentList {
	return &PgStepValueCurrentList{}
}

// Exec returns one latest Step Value per key, ordered by key.
func (operation *PgStepValueCurrentList) Exec(
	ctx context.Context,
	request *StepValueCurrentListRequest,
) ([]*StepValue, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgStepValueCurrentList")
	defer span.End()

	span.SetAttributes(attribute.String("project.id", request.ProjectID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	values := make([]*StepValue, 0)

	err = db.NewRaw(stepValueCurrentListQuery, request.ProjectID).Scan(ctx, &values)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, values), nil
}
