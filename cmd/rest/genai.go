package main

import (
	"context"
	"net"
	"strconv"

	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

// genaiClient is what the generation services call on genai.
type genaiClient interface {
	core.GenerationSubmitServiceGenai
	core.GenerationGetServiceGenai
	core.GenerationCancelServiceGenai
	core.GenerationStreamServiceGenai
	handlers.RestHealthApiGenai

	Close()
}

// errGenaiUnconfigured answers every call while no genai host is configured.
var errGenaiUnconfigured = status.Error(codes.Unavailable, "genai is not configured")

// newGenaiClient connects to genai. Without a configured host, it returns a client whose every call
// answers that genai is unavailable, so the service runs and its generation routes answer 503.
func newGenaiClient(ctx context.Context, dependencies config.Dependencies) genaiClient {
	if dependencies.ServiceGenaiHost == "" {
		return unconfiguredGenai{}
	}

	return lo.Must(servicegenai.NewClient(
		net.JoinHostPort(dependencies.ServiceGenaiHost, strconv.Itoa(dependencies.ServiceGenaiPort)),
		lo.Must(dependencies.ServiceGenaiCredentials.Options(ctx))...,
	))
}

// genaiHealth is what the health check probes on genai: the client, or nil while no genai host is
// configured, which leaves genai out of the report.
func genaiHealth(client genaiClient, dependencies config.Dependencies) handlers.RestHealthApiGenai {
	if dependencies.ServiceGenaiHost == "" {
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

func (unconfiguredGenai) Status(
	context.Context, *servicegenai.StatusRequest, ...grpc.CallOption,
) (*servicegenai.StatusResponse, error) {
	return nil, errGenaiUnconfigured
}

func (unconfiguredGenai) Close() {}
