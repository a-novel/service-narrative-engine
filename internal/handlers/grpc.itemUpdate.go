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

// GrpcItemUpdateService is the core operation [GrpcItemUpdate] delegates to.
type GrpcItemUpdateService interface {
	Exec(ctx context.Context, request *core.ItemUpdateRequest) (*core.Item, error)
}

// GrpcItemUpdate is the gRPC handler for the ItemUpdate RPC.
type GrpcItemUpdate struct {
	templatev0.UnimplementedItemUpdateServiceServer

	service GrpcItemUpdateService
}

func NewGrpcItemUpdate(service GrpcItemUpdateService) *GrpcItemUpdate {
	return &GrpcItemUpdate{service: service}
}

// ItemUpdate replaces the name and description of the item with the given ID and returns the
// updated record.
func (handler *GrpcItemUpdate) ItemUpdate(
	ctx context.Context, request *templatev0.ItemUpdateRequest,
) (*templatev0.ItemUpdateResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.ItemUpdate")
	defer span.End()

	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid item id")
	}

	serviceRequest := &core.ItemUpdateRequest{
		ID:          id,
		Name:        request.GetName(),
		Description: request.GetDescription(),
	}

	item, err := handler.service.Exec(ctx, serviceRequest)
	if errors.Is(err, core.ErrInvalidRequest) {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if errors.Is(err, core.ErrItemUpdateNotFound) {
		return nil, status.Error(codes.NotFound, "item not found")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &templatev0.ItemUpdateResponse{Item: itemToProto(item)}, nil
}
