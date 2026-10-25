package handlers

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-template/internal/core"
	templatev0 "github.com/a-novel/service-template/internal/handlers/protogen/anovel/template/v0"
)

// GrpcItemDeleteService is the core operation [GrpcItemDelete] delegates to.
type GrpcItemDeleteService interface {
	Exec(ctx context.Context, request *core.ItemDeleteRequest) (*core.Item, error)
}

// GrpcItemDelete is the gRPC handler for the ItemDelete RPC.
type GrpcItemDelete struct {
	templatev0.UnimplementedItemDeleteServiceServer

	service GrpcItemDeleteService
}

func NewGrpcItemDelete(service GrpcItemDeleteService) *GrpcItemDelete {
	return &GrpcItemDelete{service: service}
}

// ItemDelete deletes the item with the given ID and returns the removed record.
func (handler *GrpcItemDelete) ItemDelete(
	ctx context.Context, request *templatev0.ItemDeleteRequest,
) (*templatev0.ItemDeleteResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.ItemDelete")
	defer span.End()

	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid item id")
	}

	item, err := handler.service.Exec(ctx, &core.ItemDeleteRequest{ID: id})
	if errors.Is(err, core.ErrInvalidRequest) {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if errors.Is(err, core.ErrItemDeleteNotFound) {
		return nil, status.Error(codes.NotFound, "item not found")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &templatev0.ItemDeleteResponse{Item: itemToProto(item)}, nil
}
