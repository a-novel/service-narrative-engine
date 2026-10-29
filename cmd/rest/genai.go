package main

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
)

// genaiClient is what the generation services call on genai.
type genaiClient interface {
	core.GenerationSubmitServiceGenai
	core.GenerationGetServiceGenai
	core.GenerationCancelServiceGenai

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
		fmt.Sprintf("%s:%d", dependencies.ServiceGenaiHost, dependencies.ServiceGenaiPort),
		lo.Must(dependencies.ServiceGenaiCredentials.Options(ctx))...,
	))
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

func (unconfiguredGenai) Close() {}
