package handlers

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/handlers/protogen"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

type ItemUpdateService interface {
	Exec(ctx context.Context, request *services.ItemUpdateRequest) (*services.Item, error)
}

type ItemUpdate struct {
	protogen.UnimplementedItemUpdateServiceServer

	service ItemUpdateService
}

func NewItemUpdate(service ItemUpdateService) *ItemUpdate {
	return &ItemUpdate{service: service}
}

func (handler *ItemUpdate) ItemUpdate(
	ctx context.Context, request *protogen.ItemUpdateRequest,
) (*protogen.ItemUpdateResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.ItemUpdate")
	defer span.End()

	id, err := uuid.Parse(request.GetId())
	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.InvalidArgument, "invalid item id")
	}

	item, err := handler.service.Exec(ctx, &services.ItemUpdateRequest{
		ID:          id,
		Name:        request.GetName(),
		Description: request.GetDescription(),
	})
	if errors.Is(err, services.ErrInvalidRequest) {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if errors.Is(err, dao.ErrItemUpdateNotFound) {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.NotFound, "item not found")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &protogen.ItemUpdateResponse{Item: itemToProto(item)}, nil
}
