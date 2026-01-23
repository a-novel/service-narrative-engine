package handlers

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-template/internal/handlers/protogen"
	"github.com/a-novel/service-template/internal/services"
)

type ItemListService interface {
	Exec(ctx context.Context, request *services.ItemListRequest) ([]*services.Item, error)
}

type ItemList struct {
	protogen.UnimplementedItemListServiceServer

	service ItemListService
}

func NewItemList(service ItemListService) *ItemList {
	return &ItemList{service: service}
}

func (handler *ItemList) ItemList(
	ctx context.Context, request *protogen.ItemListRequest,
) (*protogen.ItemListResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.ItemList")
	defer span.End()

	items, err := handler.service.Exec(ctx, &services.ItemListRequest{
		Limit:  int(request.GetLimit()),
		Offset: int(request.GetOffset()),
	})
	if errors.Is(err, services.ErrInvalidRequest) {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	protoItems := make([]*protogen.Item, len(items))
	for i, item := range items {
		protoItems[i] = itemToProto(item)
	}

	return &protogen.ItemListResponse{Items: protoItems}, nil
}
