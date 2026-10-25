package handlers

import (
	"context"
	"errors"

	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-template/internal/core"
	templatev0 "github.com/a-novel/service-template/internal/handlers/protogen/anovel/template/v0"
)

// GrpcItemListService is the core operation [GrpcItemList] delegates to.
type GrpcItemListService interface {
	Exec(ctx context.Context, request *core.ItemListRequest) ([]*core.Item, error)
}

// GrpcItemList is the gRPC handler for the ItemList RPC.
type GrpcItemList struct {
	templatev0.UnimplementedItemListServiceServer

	service GrpcItemListService
}

func NewGrpcItemList(service GrpcItemListService) *GrpcItemList {
	return &GrpcItemList{service: service}
}

// ItemList returns a page of items, newest first.
func (handler *GrpcItemList) ItemList(
	ctx context.Context, request *templatev0.ItemListRequest,
) (*templatev0.ItemListResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.ItemList")
	defer span.End()

	after, err := parseOptionalItemID(request.GetAfter())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid item id")
	}

	serviceRequest := &core.ItemListRequest{
		Limit: int(request.GetLimit()),
		After: after,
	}

	items, err := handler.service.Exec(ctx, serviceRequest)
	if errors.Is(err, core.ErrInvalidRequest) {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &templatev0.ItemListResponse{
		Items: lo.Map(items, func(item *core.Item, _ int) *templatev0.Item { return itemToProto(item) }),
	}, nil
}
