package handlers

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-template/internal/core"
	templatev0 "github.com/a-novel/service-template/internal/handlers/protogen/anovel/template/v0"
)

// GrpcItemCreateService is the core operation [GrpcItemCreate] delegates to.
type GrpcItemCreateService interface {
	Exec(ctx context.Context, request *core.ItemCreateRequest) (*core.Item, error)
}

// GrpcItemCreate is the gRPC handler for the ItemCreate RPC.
type GrpcItemCreate struct {
	templatev0.UnimplementedItemCreateServiceServer

	service GrpcItemCreateService
}

func NewGrpcItemCreate(service GrpcItemCreateService) *GrpcItemCreate {
	return &GrpcItemCreate{service: service}
}

// ItemCreate creates an item and returns it.
func (handler *GrpcItemCreate) ItemCreate(
	ctx context.Context, request *templatev0.ItemCreateRequest,
) (*templatev0.ItemCreateResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.ItemCreate")
	defer span.End()

	serviceRequest := &core.ItemCreateRequest{
		Name:        request.GetName(),
		Description: request.GetDescription(),
	}

	item, err := handler.service.Exec(ctx, serviceRequest)
	if errors.Is(err, core.ErrInvalidRequest) {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &templatev0.ItemCreateResponse{Item: itemToProto(item)}, nil
}
