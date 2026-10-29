package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"
	"google.golang.org/grpc"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

const (
	// RestHealthStatusUp marks a dependency the service can currently reach.
	RestHealthStatusUp = "up"
	// RestHealthStatusDown marks a dependency the service failed to reach.
	RestHealthStatusDown = "down"
)

// RestHealthStatus is the JSON representation of a single dependency's health.
// /v1/healthcheck is unauthenticated, so the body carries the state alone. A raw error
// message carries internal hostnames, ports, or schema names. The underlying error
// goes to the trace span, where operators can read it.
type RestHealthStatus struct {
	// Status is either [RestHealthStatusUp] or [RestHealthStatusDown].
	Status string `json:"status"`
}

// NewRestHealthStatus converts an error into a RestHealthStatus, mapping nil to
// [RestHealthStatusUp] and any non-nil error to [RestHealthStatusDown]. The error
// itself is dropped from the public response; see [RestHealthStatus].
func NewRestHealthStatus(err error) *RestHealthStatus {
	return &RestHealthStatus{
		Status: lo.Ternary(err == nil, RestHealthStatusUp, RestHealthStatusDown),
	}
}

// RestHealthApiJsonKeys is the JSON Keys client surface the health check probes.
type RestHealthApiJsonKeys interface {
	Status(
		ctx context.Context,
		req *servicejsonkeys.StatusRequest,
		opts ...grpc.CallOption,
	) (*servicejsonkeys.StatusResponse, error)
}

// RestHealth is the HTTP handler for the health endpoint. It probes each
// backing dependency and reports their individual status, with HTTP 503 when any
// probe fails.
type RestHealth struct {
	apiJsonKeys RestHealthApiJsonKeys
}

func NewRestHealth(apiJsonKeys RestHealthApiJsonKeys) *RestHealth {
	return &RestHealth{apiJsonKeys: apiJsonKeys}
}

func (handler *RestHealth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.Health")
	defer span.End()

	statusCode := http.StatusOK
	report := map[string]any{}

	err := handler.reportPostgres(ctx)
	if err != nil {
		_ = otel.ReportError(span, err)
		statusCode = http.StatusServiceUnavailable
	}

	report["client:postgres"] = NewRestHealthStatus(err)

	err = handler.reportJsonKeys(ctx)
	if err != nil {
		_ = otel.ReportError(span, err)
		statusCode = http.StatusServiceUnavailable
	}

	report["api:jsonKeys"] = NewRestHealthStatus(err)

	httpf.SendJSONStatus(w, span, statusCode, report)
}

func (handler *RestHealth) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(reportPostgres)")
	defer span.End()

	err := postgres.Health(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	return nil
}

func (handler *RestHealth) reportJsonKeys(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(reportJsonKeys)")
	defer span.End()

	// JSON Keys fails Status with Unavailable when one of its own dependencies is down, so its
	// success is the whole verdict.
	_, err := handler.apiJsonKeys.Status(ctx, &servicejsonkeys.StatusRequest{})
	if err != nil {
		return otel.ReportError(span, err)
	}

	return nil
}
