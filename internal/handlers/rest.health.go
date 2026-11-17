package handlers

import (
	"context"
	"net/http"

	"google.golang.org/grpc"

	servicegenai "github.com/a-novel/service-genai/pkg/go"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/postgres"
)

// RestHealthAPIJSONKeys is the JSON Keys client surface the health check probes.
type RestHealthAPIJSONKeys interface {
	Status(
		ctx context.Context,
		req *servicejsonkeys.StatusRequest,
		opts ...grpc.CallOption,
	) (*servicejsonkeys.StatusResponse, error)
}

// RestHealthAPIGenai is the genai client surface the health check probes.
type RestHealthAPIGenai interface {
	Status(
		ctx context.Context,
		req *servicegenai.StatusRequest,
		opts ...grpc.CallOption,
	) (*servicegenai.StatusResponse, error)
}

// NewRestHealth returns the handler of the public health route, which reports PostgreSQL, the JSON
// Keys service and genai as up or down, as [httpf.Health] answers. A nil apiGenai, as while no genai
// host is configured, leaves genai out of the report.
func NewRestHealth(apiJSONKeys RestHealthAPIJSONKeys, apiGenai RestHealthAPIGenai) http.HandlerFunc {
	probes := map[string]func(ctx context.Context) error{
		"client:postgres": postgres.Health,
		// JSON Keys fails Status with Unavailable when one of its own dependencies is down, so its
		// success is the whole verdict.
		"api:jsonKeys": func(ctx context.Context) error {
			_, err := apiJSONKeys.Status(ctx, &servicejsonkeys.StatusRequest{})

			return err
		},
	}

	if apiGenai != nil {
		// genai fails Status with Unavailable when its database is down, so its success is the whole
		// verdict. It never calls a provider.
		probes["api:genai"] = func(ctx context.Context) error {
			_, err := apiGenai.Status(ctx, &servicegenai.StatusRequest{})

			return err
		}
	}

	return httpf.Health(probes)
}
