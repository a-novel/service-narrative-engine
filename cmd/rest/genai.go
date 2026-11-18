package main

import (
	"context"

	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"

	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

// genaiClient is what the generation services call on genai.
type genaiClient interface {
	core.GenerationSubmitServiceGenai
	core.GenerationGetServiceGenai
	core.GenerationCancelServiceGenai
	core.GenerationStreamServiceGenai
	core.GenerationUsageServiceGenai
	handlers.RestHealthAPIGenai

	Close()
}

// errGenaiUnconfigured answers every call while no genai host is configured.
var errGenaiUnconfigured = status.Error(codes.Unavailable, "genai is not configured")

// newGenaiClient connects to genai. Without a configured host, it returns a client whose every call
// answers that genai is unavailable, so the service runs and its generation routes answer 503.
func newGenaiClient(genai grpcf.ClientConfig) genaiClient {
	if genai.Host == "" {
		return unconfiguredGenai{}
	}

	return lo.Must(servicegenai.NewClient(genai.Address(), genai.DialOptions()...))
}

// genaiHealth is what the health check probes on genai: the client, or nil while no genai host is
// configured, which leaves genai out of the report.
func genaiHealth(client genaiClient, genai grpcf.ClientConfig) handlers.RestHealthAPIGenai {
	if genai.Host == "" {
		return nil
	}

	return client
}

// unconfiguredGenai stands in for genai while no host is configured.
type unconfiguredGenai struct{}

func (unconfiguredGenai) GenerationSubmit(
	context.Context, *servicegenai.GenerationSubmitRequest, ...grpc.CallOption,
) (*servicegenai.GenerationSubmitResponse, error) {
	return nil, errGenaiUnconfigured
}

func (unconfiguredGenai) GenerationGet(
	context.Context, *servicegenai.GenerationGetRequest, ...grpc.CallOption,
) (*servicegenai.GenerationGetResponse, error) {
	return nil, errGenaiUnconfigured
}

func (unconfiguredGenai) GenerationCancel(
	context.Context, *servicegenai.GenerationCancelRequest, ...grpc.CallOption,
) (*servicegenai.GenerationCancelResponse, error) {
	return nil, errGenaiUnconfigured
}

func (unconfiguredGenai) GenerationStream(
	context.Context, *servicegenai.GenerationStreamRequest, ...grpc.CallOption,
) (grpc.ServerStreamingClient[servicegenai.GenerationStreamResponse], error) {
	return nil, errGenaiUnconfigured
}

func (unconfiguredGenai) UsageSummary(
	context.Context, *servicegenai.UsageSummaryRequest, ...grpc.CallOption,
) (*servicegenai.UsageSummaryResponse, error) {
	return nil, errGenaiUnconfigured
}

func (unconfiguredGenai) Status(
	context.Context, *servicegenai.StatusRequest, ...grpc.CallOption,
) (*servicegenai.StatusResponse, error) {
	return nil, errGenaiUnconfigured
}

func (unconfiguredGenai) Close() {}
