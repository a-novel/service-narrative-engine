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
)

var errStepValueMissing = errors.New("step value insert returned no entity")

// StepValueInsertDao persists source-agnostic content for one Engine step.
type StepValueInsertDao interface {
	Exec(ctx context.Context, request *dao.StepValueInsertRequest) (*dao.StepValue, error)
}

// StepValueCreateRequest carries only the client-composed content for one step.
type StepValueCreateRequest struct {
	Actor           Actor           `validate:"actor"`
	IdeaID          uuid.UUID       `validate:"required"`
	EngineVersionID uuid.UUID       `validate:"required"`
	StepKey         string          `validate:"required,notblank,max=256"`
	Value           json.RawMessage `validate:"required"`
}

// StepValueCreate validates and saves one independent project step.
type StepValueCreate struct {
	projectAccess    ProjectAccessService
	engineVersionDao EngineVersionSelectDao
	dao              StepValueInsertDao
	transactor       transaction.Transactor
}

// NewStepValueCreate creates an independent step-value save service.
func NewStepValueCreate(
	projectAccess ProjectAccessService,
	engineVersionDao EngineVersionSelectDao,
	stepValueDao StepValueInsertDao,
	transactor transaction.Transactor,
) *StepValueCreate {
	return &StepValueCreate{
		projectAccess:    projectAccess,
		engineVersionDao: engineVersionDao,
		dao:              stepValueDao,
		transactor:       transactor,
	}
}

// Exec saves partial content that conforms to its Engine step shape.
func (service *StepValueCreate) Exec(
	ctx context.Context,
	request *StepValueCreateRequest,
) (json.RawMessage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.StepValueCreate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("idea.id", request.IdeaID.String()),
		attribute.String("engine_version.id", request.EngineVersionID.String()),
		attribute.String("engine.step_key", request.StepKey),
		attribute.String("step_value.owner_id", request.Actor.UserID.String()),
	)

	_, err = service.projectAccess.Exec(ctx, &ProjectAccessRequest{
		Actor:  request.Actor,
		IdeaID: request.IdeaID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("access project: %w", err))
	}

	engineVersion, err := service.engineVersionDao.Exec(ctx, &dao.EngineVersionSelectRequest{
		ID: request.EngineVersionID,
	})
	if errors.Is(err, dao.ErrEngineVersionSelectNotFound) {
		err = errors.Join(err, ErrEngineVersionNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select Engine Version: %w", err))
	}

	step, err := selectEngineStep(engineVersion.Definition, request.StepKey)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	schema, err := step.loadOutputSchema()
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = schema.validatePartial(request.Value)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("%w: step value: %w", ErrInvalidRequest, err))
	}

	var entity *dao.StepValue

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		entity, err = service.dao.Exec(ctx, &dao.StepValueInsertRequest{
			ID:              uuid.Must(uuid.NewV7()),
			IdeaID:          request.IdeaID,
			OwnerID:         request.Actor.UserID,
			EngineVersionID: request.EngineVersionID,
			StepKey:         request.StepKey,
			Value:           request.Value,
			Now:             time.Now(),
		})
		if errors.Is(err, dao.ErrIdeaLockNotFound) {
			err = errors.Join(err, ErrIdeaNotFound)
		}

		if err != nil {
			return fmt.Errorf("insert step value: %w", err)
		}

		if entity == nil {
			return fmt.Errorf("insert step value: %w", errStepValueMissing)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("save step value: %w", err))
	}

	span.SetAttributes(attribute.String("step_value.id", entity.ID.String()))

	return otel.ReportSuccess(span, entity.Value), nil
}
