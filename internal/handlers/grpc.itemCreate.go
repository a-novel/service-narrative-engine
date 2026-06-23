package handlers

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/handlers/protogen"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

type ItemCreateService interface {
	Exec(ctx context.Context, request *services.ItemCreateRequest) (*services.Item, error)
}

type ItemCreate struct {
	protogen.UnimplementedItemCreateServiceServer

	service ItemCreateService
}

func NewItemCreate(service ItemCreateService) *ItemCreate {
	return &ItemCreate{service: service}
}

func (handler *ItemCreate) ItemCreate(
	ctx context.Context, request *protogen.ItemCreateRequest,
) (*protogen.ItemCreateResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.ItemCreate")
	defer span.End()

	item, err := handler.service.Exec(ctx, &services.ItemCreateRequest{
		Name:        request.GetName(),
		Description: request.GetDescription(),
	})
	if errors.Is(err, services.ErrInvalidRequest) {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &protogen.ItemCreateResponse{Item: itemToProto(item)}, nil
}

func itemToProto(item *services.Item) *protogen.Item {
	return &protogen.Item{
		Id:          item.ID.String(),
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
	}
}
