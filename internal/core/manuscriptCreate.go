package core

import (
	"bytes"
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

// ErrStepValueConflict reports content already saved for the same immutable step.
var ErrStepValueConflict = errors.New("step value already exists")

var errManuscriptInsertMissing = errors.New("manuscript insert returned no entity")

// StepValueInsertDao persists source-agnostic content for one Engine step.
type StepValueInsertDao interface {
	Exec(ctx context.Context, request *dao.StepValueInsertRequest) (*dao.StepValue, error)
}

// ManuscriptInsertDao persists a self-contained Manuscript.
type ManuscriptInsertDao interface {
	Exec(ctx context.Context, request *dao.ManuscriptInsertRequest) (*dao.Manuscript, error)
}

// ManuscriptCreateRequest carries client-composed step content and an opaque Manuscript.
type ManuscriptCreateRequest struct {
	Actor           Actor           `validate:"required"`
	IdeaID          uuid.UUID       `validate:"required"`
	EngineVersionID uuid.UUID       `validate:"required"`
	StepKey         string          `validate:"required,notblank,max=256"`
	StepValue       json.RawMessage `validate:"required,max=1048576"`
	Manuscript      json.RawMessage `validate:"required"`
}

// ManuscriptCreate validates and saves a step value plus Manuscript atomically.
type ManuscriptCreate struct {
	ideaDao          IdeaSelectDao
	engineVersionDao EngineVersionSelectDao
	stepValueDao     StepValueInsertDao
	manuscriptDao    ManuscriptInsertDao
	transactor       transaction.Transactor
}

// NewManuscriptCreate creates the explicit project-content save service.
func NewManuscriptCreate(
	ideaDao IdeaSelectDao,
	engineVersionDao EngineVersionSelectDao,
	stepValueDao StepValueInsertDao,
	manuscriptDao ManuscriptInsertDao,
	transactor transaction.Transactor,
) *ManuscriptCreate {
	return &ManuscriptCreate{
		ideaDao:          ideaDao,
		engineVersionDao: engineVersionDao,
		stepValueDao:     stepValueDao,
		manuscriptDao:    manuscriptDao,
		transactor:       transactor,
	}
}

// Exec saves client-selected content without generation provenance.
func (service *ManuscriptCreate) Exec(
	ctx context.Context,
	request *ManuscriptCreateRequest,
) (json.RawMessage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ManuscriptCreate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	manuscriptJSON := bytes.TrimSpace(request.Manuscript)
	if !json.Valid(manuscriptJSON) {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: manuscript must contain valid JSON", ErrInvalidRequest),
		)
	}

	if manuscriptJSON[0] != '{' {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: manuscript must be a JSON object", ErrInvalidRequest),
		)
	}

	span.SetAttributes(
		attribute.String("idea.id", request.IdeaID.String()),
		attribute.String("engine_version.id", request.EngineVersionID.String()),
		attribute.String("engine.step_key", request.StepKey),
		attribute.String("manuscript.owner_id", request.Actor.UserID.String()),
	)

	_, err = service.ideaDao.Exec(ctx, &dao.IdeaSelectRequest{
		ID:      request.IdeaID,
		OwnerID: request.Actor.UserID,
	})
	if err != nil {
		if errors.Is(err, dao.ErrIdeaSelectNotFound) {
			err = errors.Join(err, ErrIdeaNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("select Idea: %w", err))
	}

	engineVersion, err := service.engineVersionDao.Exec(ctx, &dao.EngineVersionSelectRequest{
		ID: request.EngineVersionID,
	})
	if err != nil {
		if errors.Is(err, dao.ErrEngineVersionSelectNotFound) {
			err = errors.Join(err, ErrEngineVersionNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("select Engine Version: %w", err))
	}

	step, err := selectEngineStep(engineVersion.Definition, request.StepKey)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = step.validateValue(request.StepValue)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("%w: step value: %w", ErrInvalidRequest, err))
	}

	var entity *dao.Manuscript

	now := time.Now()
	stepValueID := uuid.Must(uuid.NewV7())
	manuscriptID := uuid.Must(uuid.NewV7())

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, insertErr := service.stepValueDao.Exec(ctx, &dao.StepValueInsertRequest{
			ID:              stepValueID,
			IdeaID:          request.IdeaID,
			EngineVersionID: request.EngineVersionID,
			StepKey:         request.StepKey,
			Value:           request.StepValue,
			Now:             now,
		})
		if insertErr != nil {
			return fmt.Errorf("insert step value: %w", insertErr)
		}

		entity, insertErr = service.manuscriptDao.Exec(ctx, &dao.ManuscriptInsertRequest{
			ID:     manuscriptID,
			IdeaID: request.IdeaID,
			Value:  request.Manuscript,
			Now:    now,
		})
		if insertErr != nil {
			return fmt.Errorf("insert Manuscript: %w", insertErr)
		}

		return nil
	})
	if errors.Is(err, dao.ErrStepValueInsertConflict) {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %w", ErrStepValueConflict, err))
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("save project content: %w", err))
	}

	if entity == nil {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("save project content: %w", errManuscriptInsertMissing),
		)
	}

	span.SetAttributes(attribute.String("manuscript.id", entity.ID.String()))

	return otel.ReportSuccess(span, entity.Value), nil
}
