package handlers

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	templatev0 "github.com/a-novel/service-template/internal/handlers/protogen/anovel/template/v0"
)

// NewGrpcHealthStatus returns the health report for a dependency whose probe succeeded. A failed
// probe answers Unavailable instead, so the report never carries a raw dependency error.
func NewGrpcHealthStatus() *templatev0.DependencyHealth {
	return &templatev0.DependencyHealth{
		Status: templatev0.DependencyStatus_DEPENDENCY_STATUS_UP,
	}
}

// GrpcStatus is the gRPC handler for the Status RPC, reporting the health of the
// service's external dependencies.
type GrpcStatus struct {
	templatev0.UnimplementedStatusServiceServer
}

func NewGrpcStatus() *GrpcStatus {
	return &GrpcStatus{}
}

// Status probes each dependency and returns their health, or Unavailable when a probe fails.
func (handler *GrpcStatus) Status(
	ctx context.Context, _ *templatev0.StatusRequest,
) (*templatev0.StatusResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.Status")
	defer span.End()

	err := handler.reportPostgres(ctx)
	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Unavailable, "service dependencies unavailable")
	}

	return &templatev0.StatusResponse{
		Postgres: NewGrpcHealthStatus(),
	}, nil
}

func (handler *GrpcStatus) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "grpc.Status(reportPostgres)")
	defer span.End()

	err := postgres.Health(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	return nil
}
