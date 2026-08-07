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

// ManuscriptInsertDao persists one validated Manuscript Version.
type ManuscriptInsertDao interface {
	Exec(ctx context.Context, request *dao.ManuscriptInsertRequest) (*dao.Manuscript, error)
}

// ManuscriptCreateRequest carries one complete static Manuscript document.
type ManuscriptCreateRequest struct {
	Actor      Actor           `validate:"actor"`
	ProjectID  uuid.UUID       `validate:"required"`
	Manuscript json.RawMessage `validate:"required"`
}

// Manuscript is one saved static document version.
type Manuscript struct {
	ID         uuid.UUID       `json:"id"`
	ProjectID  uuid.UUID       `json:"projectID"`
	Manuscript json.RawMessage `json:"manuscript"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// manuscriptFromDao maps one stored document into the core contract.
func manuscriptFromDao(entity *dao.Manuscript) *Manuscript {
	if entity == nil {
		return nil
	}

	return &Manuscript{
		ID:         entity.ID,
		ProjectID:  entity.ProjectID,
		Manuscript: entity.Value,
		CreatedAt:  entity.CreatedAt,
	}
}

// ManuscriptCreate validates and saves one Project-owned Manuscript Version.
type ManuscriptCreate struct {
	projectAccess ProjectAccessService
	dao           ManuscriptInsertDao
	transactor    transaction.Transactor
}

// NewManuscriptCreate creates a Project-owned Manuscript save service.
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

// Exec validates the static contract and appends a Manuscript Version.
func (service *ManuscriptCreate) Exec(
	ctx context.Context,
	request *ManuscriptCreateRequest,
) (*Manuscript, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ManuscriptCreate")
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

	err = manuscriptContentDefinition.validate(request.Manuscript)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("manuscript: %w", err))
	}

	var entity *dao.Manuscript

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		entity, err = service.dao.Exec(ctx, &dao.ManuscriptInsertRequest{
			ID:        uuid.Must(uuid.NewV7()),
			ProjectID: request.ProjectID,
			OwnerID:   request.Actor.UserID,
			Value:     request.Manuscript,
			Now:       time.Now(),
		})
		if errors.Is(err, dao.ErrProjectLockNotFound) {
			err = errors.Join(err, ErrProjectNotFound)
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

	return otel.ReportSuccess(span, manuscriptFromDao(entity)), nil
}
