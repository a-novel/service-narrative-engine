package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// IdeaCreateDao persists Ideas created by [IdeaCreate].
type IdeaCreateDao interface {
	Exec(ctx context.Context, request *dao.IdeaInsertRequest) (*dao.Idea, error)
}

// IdeaCreateRequest carries the authenticated owner and typed Idea fields.
type IdeaCreateRequest struct {
	Actor Actor  `validate:"required"`
	Seed  string `validate:"required,notblank,maxbytes=65536"`
	Genre string `validate:"required,notblank,max=256"`
	Title string `validate:"max=256"`
}

// IdeaCreate validates and persists a new Idea.
type IdeaCreate struct {
	dao IdeaCreateDao
}

// NewIdeaCreate creates an Idea service.
func NewIdeaCreate(ideaDao IdeaCreateDao) *IdeaCreate {
	return &IdeaCreate{dao: ideaDao}
}

// Exec creates an Idea owned by the authenticated actor.
func (service *IdeaCreate) Exec(ctx context.Context, request *IdeaCreateRequest) (*Idea, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.IdeaCreate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(attribute.String("idea.owner_id", request.Actor.UserID.String()))

	entity, err := service.dao.Exec(ctx, &dao.IdeaInsertRequest{
		ID:      uuid.Must(uuid.NewV7()),
		OwnerID: request.Actor.UserID,
		Seed:    request.Seed,
		Genre:   request.Genre,
		Title:   request.Title,
		Now:     time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("insert idea: %w", err))
	}

	span.SetAttributes(attribute.String("idea.id", entity.ID.String()))

	return otel.ReportSuccess(span, &Idea{
		ID:        entity.ID,
		OwnerID:   entity.OwnerID,
		Seed:      entity.Seed,
		Genre:     entity.Genre,
		Title:     entity.Title,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}), nil
}
