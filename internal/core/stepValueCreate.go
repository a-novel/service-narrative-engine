package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

var errStepValueMissing = errors.New("step value insert returned no entity")

// StepValueInsertDao persists one opaque content version.
type StepValueInsertDao interface {
	Exec(ctx context.Context, request *dao.StepValueInsertRequest) (*dao.StepValue, error)
}

// StepValueCreateRequest carries arbitrary JSON under one Project-owned client key.
type StepValueCreateRequest struct {
	Actor     Actor           `validate:"actor"`
	ProjectID uuid.UUID       `validate:"required"`
	Key       string          `validate:"required,notblank,max=256"`
	Value     json.RawMessage `validate:"required"`
}

// StepValue is one saved opaque JSON version.
type StepValue struct {
	ID        uuid.UUID       `json:"id"`
	ProjectID uuid.UUID       `json:"projectID"`
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	CreatedAt time.Time       `json:"createdAt"`
}

// stepValueFromDao maps one stored opaque value into the core contract.
func stepValueFromDao(entity *dao.StepValue) *StepValue {
	if entity == nil {
		return nil
	}

	return &StepValue{
		ID:        entity.ID,
		ProjectID: entity.ProjectID,
		Key:       entity.Key,
		Value:     entity.Value,
		CreatedAt: entity.CreatedAt,
	}
}

// StepValueCreate saves Project-owned JSON without interpreting its shape.
type StepValueCreate struct {
	projectAccess ProjectAccessService
	dao           StepValueInsertDao
	transactor    transaction.Transactor
}

// NewStepValueCreate creates an opaque Step Value save service.
func NewStepValueCreate(
	projectAccess ProjectAccessService,
	stepValueDao StepValueInsertDao,
	transactor transaction.Transactor,
) *StepValueCreate {
	return &StepValueCreate{
		projectAccess: projectAccess,
		dao:           stepValueDao,
		transactor:    transactor,
	}
}

// Exec appends arbitrary bounded JSON under an opaque key.
func (service *StepValueCreate) Exec(
	ctx context.Context,
	request *StepValueCreateRequest,
) (*StepValue, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.StepValueCreate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.Actor.UserID.String()),
	)

	_, err = service.projectAccess.Exec(ctx, &ProjectAccessRequest{
		Actor:     request.Actor,
		ProjectID: request.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("access Project: %w", err))
	}

	err = lib.ValidateJSON(request.Value, schemas.ContentDocumentMaxBytes)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("step value: %w", errors.Join(ErrInvalidRequest, err)))
	}

	var entity *dao.StepValue

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		entity, err = service.dao.Exec(ctx, &dao.StepValueInsertRequest{
			ID:        uuid.Must(uuid.NewV7()),
			ProjectID: request.ProjectID,
			OwnerID:   request.Actor.UserID,
			Key:       request.Key,
			Value:     request.Value,
			Now:       time.Now(),
		})
		if errors.Is(err, dao.ErrProjectLockNotFound) {
			err = errors.Join(err, ErrProjectNotFound)
		}

		if err != nil {
			return fmt.Errorf("insert Step Value: %w", err)
		}

		if entity == nil {
			return fmt.Errorf("insert Step Value: %w", errStepValueMissing)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("save Step Value: %w", err))
	}

	span.SetAttributes(attribute.String("step_value.id", entity.ID.String()))

	return otel.ReportSuccess(span, stepValueFromDao(entity)), nil
}
