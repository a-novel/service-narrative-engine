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

type ItemCreateRepository interface {
	Exec(ctx context.Context, request *dao.ItemCreateRequest) (*dao.Item, error)
}

type ItemCreateRequest struct {
	Name        string `validate:"required,notblank,max=256"`
	Description string `validate:"max=1024"`
}

// ItemCreate validates and inserts a new item.
type ItemCreate struct {
	repository ItemCreateRepository
}

func NewItemCreate(repository ItemCreateRepository) *ItemCreate {
	return &ItemCreate{repository: repository}
}

func (service *ItemCreate) Exec(ctx context.Context, request *ItemCreateRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ItemCreate")
	defer span.End()

	span.SetAttributes(attribute.String("item.name", request.Name))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.repository.Exec(ctx, &dao.ItemCreateRequest{
		ID:          uuid.New(),
		Name:        request.Name,
		Description: request.Description,
		Now:         time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create item: %w", err))
	}

	return otel.ReportSuccess(span, &Item{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}), nil
}
