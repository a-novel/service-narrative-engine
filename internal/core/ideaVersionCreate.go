package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var errIdeaVersionInsertMissing = errors.New("idea version insert returned no entity")

// IdeaVersionInsertDao persists typed content under an existing Idea root.
type IdeaVersionInsertDao interface {
	Exec(ctx context.Context, request *dao.IdeaVersionInsertRequest) (*dao.IdeaVersion, error)
}

// IdeaVersionCreateRequest carries one partial Idea save under an owned project.
type IdeaVersionCreateRequest struct {
	Actor  Actor     `validate:"required"`
	IdeaID uuid.UUID `validate:"required"`
	Seed   string
	Genre  string
	Title  string
}

// IdeaVersionCreate validates and saves one immutable Idea content version.
type IdeaVersionCreate struct {
	projectAccess ProjectAccessService
	dao           IdeaVersionInsertDao
	transactor    transaction.Transactor
}

// NewIdeaVersionCreate creates an owner-scoped Idea-version service.
func NewIdeaVersionCreate(
	projectAccess ProjectAccessService,
	ideaVersionDao IdeaVersionInsertDao,
	transactor transaction.Transactor,
) *IdeaVersionCreate {
	return &IdeaVersionCreate{
		projectAccess: projectAccess,
		dao:           ideaVersionDao,
		transactor:    transactor,
	}
}

// Exec appends partial Idea content and retains the newest versions.
func (service *IdeaVersionCreate) Exec(
	ctx context.Context,
	request *IdeaVersionCreateRequest,
) (*Idea, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.IdeaVersionCreate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("idea.id", request.IdeaID.String()),
		attribute.String("idea.owner_id", request.Actor.UserID.String()),
	)

	idea, err := service.projectAccess.Exec(ctx, &ProjectAccessRequest{
		Actor:  request.Actor,
		IdeaID: request.IdeaID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("access project: %w", err))
	}

	err = validateIdeaContent(request.Seed, request.Genre, request.Title)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	var entity *dao.IdeaVersion

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		entity, err = service.dao.Exec(ctx, &dao.IdeaVersionInsertRequest{
			ID:      uuid.Must(uuid.NewV7()),
			IdeaID:  request.IdeaID,
			OwnerID: request.Actor.UserID,
			Seed:    request.Seed,
			Genre:   request.Genre,
			Title:   request.Title,
			Now:     time.Now(),
		})
		if errors.Is(err, dao.ErrIdeaLockNotFound) {
			err = errors.Join(err, ErrIdeaNotFound)
		}

		if err != nil {
			return fmt.Errorf("insert Idea version: %w", err)
		}

		if entity == nil {
			return fmt.Errorf("insert Idea version: %w", errIdeaVersionInsertMissing)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("save Idea version: %w", err))
	}

	updatedAt := entity.CreatedAt

	return otel.ReportSuccess(span, &Idea{
		ID:        idea.ID,
		OwnerID:   idea.OwnerID,
		Seed:      entity.Seed,
		Genre:     entity.Genre,
		Title:     entity.Title,
		CreatedAt: idea.CreatedAt,
		UpdatedAt: &updatedAt,
	}), nil
}
