package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-template/internal/dao"
)

type ItemUpdateRepository interface {
	Exec(ctx context.Context, request *dao.ItemUpdateRequest) (*dao.Item, error)
}

type ItemUpdateRequest struct {
	ID          uuid.UUID
	Name        string `validate:"required,notblank,max=256"`
	Description string `validate:"max=1024"`
}

// ItemUpdate validates and updates an existing item's fields.
type ItemUpdate struct {
	repository ItemUpdateRepository
}

func NewItemUpdate(repository ItemUpdateRepository) *ItemUpdate {
	return &ItemUpdate{repository: repository}
}

func (service *ItemUpdate) Exec(ctx context.Context, request *ItemUpdateRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ItemUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("item.id", request.ID.String()),
		attribute.String("item.name", request.Name),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.repository.Exec(ctx, &dao.ItemUpdateRequest{
		ID:          request.ID,
		Name:        request.Name,
		Description: request.Description,
		Now:         time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update item: %w", err))
	}

	return otel.ReportSuccess(span, &Item{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}), nil
}
