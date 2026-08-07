package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// StepValueHistoryDao retrieves retained values for one opaque key.
type StepValueHistoryDao interface {
	Exec(ctx context.Context, request *dao.StepValueListRequest) ([]*dao.StepValue, error)
}

// StepValueHistoryRequest identifies one owned Project key.
type StepValueHistoryRequest struct {
	Actor     Actor     `validate:"actor"`
	ProjectID uuid.UUID `validate:"required"`
	Key       string    `validate:"required,notblank,max=256"`
}

// StepValueHistory reads the bounded, newest-first history for one key.
type StepValueHistory struct {
	projectAccess ProjectAccessService
	dao           StepValueHistoryDao
}

// NewStepValueHistory creates an owner-scoped Step Value history service.
func NewStepValueHistory(
	projectAccess ProjectAccessService,
	historyDao StepValueHistoryDao,
) *StepValueHistory {
	return &StepValueHistory{projectAccess: projectAccess, dao: historyDao}
}

// Exec returns all retained values after checking Project ownership.
func (service *StepValueHistory) Exec(
	ctx context.Context,
	request *StepValueHistoryRequest,
) ([]*StepValue, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.StepValueHistory")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.Actor.UserID.String()),
		attribute.String("step_value.key", request.Key),
	)

	_, err = service.projectAccess.Exec(ctx, &ProjectAccessRequest{
		Actor:     request.Actor,
		ProjectID: request.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("access Project: %w", err))
	}

	entities, err := service.dao.Exec(ctx, &dao.StepValueListRequest{
		ProjectID: request.ProjectID,
		Key:       request.Key,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list Step Values: %w", err))
	}

	values := make([]*StepValue, 0, len(entities))
	for _, entity := range entities {
		values = append(values, stepValueFromDao(entity))
	}

	return otel.ReportSuccess(span, values), nil
}
