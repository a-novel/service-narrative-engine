package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"
	"github.com/uptrace/bun"
	"google.golang.org/grpc"

	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

const (
	HealthStatusUp   = "up"
	HealthStatusDown = "down"
)

type HealthStatus struct {
	Status string `json:"status"`
	Err    string `json:"err,omitempty"`
}

func NewHealthStatus(err error) *HealthStatus {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	return &HealthStatus{
		Status: lo.Ternary(err == nil, HealthStatusUp, HealthStatusDown),
		Err:    errMsg,
	}
}

type HealthApiJsonkeys interface {
	Status(ctx context.Context, req *jkpkg.StatusRequest, opts ...grpc.CallOption) (*jkpkg.StatusResponse, error)
}

type Health struct {
	apiJsonKeys HealthApiJsonkeys
}

func NewHealth(
	apiJsonKeys HealthApiJsonkeys,
) *Health {
	return &Health{
		apiJsonKeys: apiJsonKeys,
	}
}

func (handler *Health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "api.Health")
	defer span.End()

	httpf.SendJSON(ctx, w, span, map[string]any{
		"client:postgres": NewHealthStatus(handler.reportPostgres(ctx)),
		"api:jsonKeys":    NewHealthStatus(handler.reportJsonKeys(ctx)),
	})
}

func (handler *Health) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "api.Health(reportPostgres)")
	defer span.End()

	pg, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		// Cannot assess db connection if we are running on transaction mode
		return nil
	}

	err = pgdb.Ping()
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

func (handler *Health) reportJsonKeys(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "api.Health(reportJsonKeys)")
	defer span.End()

	_, err := handler.apiJsonKeys.Status(ctx, new(jkpkg.StatusRequest))
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
