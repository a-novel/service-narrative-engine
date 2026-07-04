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

// ItemCreateDao is the DAO port ItemCreate depends on to persist a new item and
// return the stored entity.
type ItemCreateDao interface {
	Exec(ctx context.Context, request *dao.ItemCreateRequest) (*dao.Item, error)
}

// ItemCreateRequest holds the validated input for [ItemCreate.Exec].
type ItemCreateRequest struct {
	Name        string `validate:"required,notblank,max=256"`
	Description string `validate:"max=1024"`
}

// ItemCreate validates and inserts a new item.
type ItemCreate struct {
	dao ItemCreateDao
}

// NewItemCreate builds an ItemCreate that persists through the given DAO.
func NewItemCreate(dao ItemCreateDao) *ItemCreate {
	return &ItemCreate{dao: dao}
}

func (service *ItemCreate) Exec(ctx context.Context, request *ItemCreateRequest) (*Item, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ItemCreate")
	defer span.End()

	span.SetAttributes(attribute.String("item.name", request.Name))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.ItemCreateRequest{
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
