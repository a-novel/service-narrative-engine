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

type ItemDeleteService interface {
	Exec(ctx context.Context, request *services.ItemDeleteRequest) (*services.Item, error)
}

type ItemDelete struct {
	protogen.UnimplementedItemDeleteServiceServer

	service ItemDeleteService
}

func NewItemDelete(service ItemDeleteService) *ItemDelete {
	return &ItemDelete{service: service}
}

func (handler *ItemDelete) ItemDelete(
	ctx context.Context, request *protogen.ItemDeleteRequest,
) (*protogen.ItemDeleteResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.ItemDelete")
	defer span.End()

	id, err := uuid.Parse(request.GetId())
	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.InvalidArgument, "invalid item id")
	}

	item, err := handler.service.Exec(ctx, &services.ItemDeleteRequest{ID: id})
	if errors.Is(err, services.ErrInvalidRequest) {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if errors.Is(err, dao.ErrItemDeleteNotFound) {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.NotFound, "item not found")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &protogen.ItemDeleteResponse{Item: itemToProto(item)}, nil
}
