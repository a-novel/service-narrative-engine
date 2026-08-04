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

var errManuscriptInsertMissing = errors.New("manuscript insert returned no entity")

// ManuscriptInsertDao persists a self-contained Manuscript.
type ManuscriptInsertDao interface {
	Exec(ctx context.Context, request *dao.ManuscriptInsertRequest) (*dao.Manuscript, error)
}

// ManuscriptCreateRequest carries only an opaque partial Manuscript document.
type ManuscriptCreateRequest struct {
	Actor      Actor           `validate:"actor"`
	IdeaID     uuid.UUID       `validate:"required"`
	Manuscript json.RawMessage `validate:"required"`
}

// ManuscriptCreate validates and saves one independent Manuscript.
type ManuscriptCreate struct {
	projectAccess ProjectAccessService
	dao           ManuscriptInsertDao
	transactor    transaction.Transactor
}

// NewManuscriptCreate creates an independent Manuscript save service.
func NewManuscriptCreate(
	projectAccess ProjectAccessService,
	manuscriptDao ManuscriptInsertDao,
	transactor transaction.Transactor,
) *ManuscriptCreate {
	return &ManuscriptCreate{
		projectAccess: projectAccess,
		dao:           manuscriptDao,
		transactor:    transactor,
	}
}

// Exec saves partial Manuscript content without generation provenance.
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

	span.SetAttributes(
		attribute.String("idea.id", request.IdeaID.String()),
		attribute.String("manuscript.owner_id", request.Actor.UserID.String()),
	)

	_, err = service.projectAccess.Exec(ctx, &ProjectAccessRequest{
		Actor:  request.Actor,
		IdeaID: request.IdeaID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("access project: %w", err))
	}

	schema, err := loadManuscriptContentSchema()
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("load Manuscript schema: %w", err))
	}

	err = schema.validatePartial(request.Manuscript)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("%w: Manuscript: %w", ErrInvalidRequest, err))
	}

	var entity *dao.Manuscript

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		entity, err = service.dao.Exec(ctx, &dao.ManuscriptInsertRequest{
			ID:      uuid.Must(uuid.NewV7()),
			IdeaID:  request.IdeaID,
			OwnerID: request.Actor.UserID,
			Value:   request.Manuscript,
			Now:     time.Now(),
		})
		if errors.Is(err, dao.ErrIdeaLockNotFound) {
			err = errors.Join(err, ErrIdeaNotFound)
		}

		if err != nil {
			return fmt.Errorf("insert Manuscript: %w", err)
		}

		if entity == nil {
			return fmt.Errorf("insert Manuscript: %w", errManuscriptInsertMissing)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("save Manuscript: %w", err))
	}

	span.SetAttributes(attribute.String("manuscript.id", entity.ID.String()))

	return otel.ReportSuccess(span, entity.Value), nil
}
