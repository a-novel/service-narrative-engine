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

// ItemCreateDao is the persistence dependency ItemCreate uses to insert an item.
type ItemCreateDao interface {
	Exec(ctx context.Context, request *dao.ItemCreateRequest) (*dao.Item, error)
}

// ItemCreateRequest carries the fields for a new item.
type ItemCreateRequest struct {
	Name        string `validate:"required,notblank,max=256"`
	Description string `validate:"max=1024"`
}

// ItemCreate validates and inserts a new item.
type ItemCreate struct {
	dao ItemCreateDao
}

func NewItemCreate(dao ItemCreateDao) *ItemCreate {
	return &ItemCreate{dao: dao}
}

func (service *ItemCreate) Exec(ctx context.Context, request *ItemCreateRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ItemCreate")
	defer span.End()

	span.SetAttributes(attribute.String("item.name", request.Name))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// A UUIDv7 sorts by creation time, which lets ItemList page by ID.
	id, err := uuid.NewV7()
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("generate item id: %w", err))
	}

	daoRequest := &dao.ItemCreateRequest{
		ID:          id,
		Name:        request.Name,
		Description: request.Description,
		Now:         time.Now(),
	}

	entity, err := service.dao.Exec(ctx, daoRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create item: %w", err))
	}

	return newItem(entity), nil
}
