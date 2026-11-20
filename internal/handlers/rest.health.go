package handlers

import (
	"context"
	"net/http"

	"google.golang.org/grpc"

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

// NewRestHealth returns the handler of the public health route, which reports PostgreSQL and the JSON
// Keys service as up or down, as [httpf.Health] answers.
func NewRestHealth(apiJSONKeys RestHealthAPIJSONKeys) http.HandlerFunc {
	return httpf.Health(map[string]func(ctx context.Context) error{
		"client:postgres": postgres.Health,
		// JSON Keys fails Status with Unavailable when one of its own dependencies is down, so its
		// success is the whole verdict.
		"api:jsonKeys": func(ctx context.Context) error {
			_, err := apiJSONKeys.Status(ctx, &servicejsonkeys.StatusRequest{})

			return err
		},
	})
}
