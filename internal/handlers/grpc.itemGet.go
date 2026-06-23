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

type ItemGetService interface {
	Exec(ctx context.Context, request *services.ItemGetRequest) (*services.Item, error)
}

type ItemGet struct {
	protogen.UnimplementedItemGetServiceServer

	service ItemGetService
}

func NewItemGet(service ItemGetService) *ItemGet {
	return &ItemGet{service: service}
}

func (handler *ItemGet) ItemGet(
	ctx context.Context, request *protogen.ItemGetRequest,
) (*protogen.ItemGetResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.ItemGet")
	defer span.End()

	id, err := uuid.Parse(request.GetId())
	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.InvalidArgument, "invalid item id")
	}

	item, err := handler.service.Exec(ctx, &services.ItemGetRequest{ID: id})
	if errors.Is(err, services.ErrInvalidRequest) {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if errors.Is(err, dao.ErrItemGetNotFound) {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.NotFound, "item not found")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &protogen.ItemGetResponse{Item: itemToProto(item)}, nil
}
