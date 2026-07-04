package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

const (
	// RestHealthStatusUp is the status reported when a dependency is reachable.
	RestHealthStatusUp = "up"
	// RestHealthStatusDown is the status reported when a dependency is unreachable.
	RestHealthStatusDown = "down"
)

// RestHealthStatus is the JSON representation of a single dependency's health.
type RestHealthStatus struct {
	// Status is either [RestHealthStatusUp] or [RestHealthStatusDown].
	Status string `json:"status"`
	// Err carries the failure message when the dependency is down; empty when it is up.
	Err string `json:"err,omitempty"`
}

// NewRestHealthStatus builds a RestHealthStatus from a dependency probe result, mapping a
// nil error to [RestHealthStatusUp] and any non-nil error to [RestHealthStatusDown].
func NewRestHealthStatus(err error) *RestHealthStatus {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	return &RestHealthStatus{
		Status: lo.Ternary(err == nil, RestHealthStatusUp, RestHealthStatusDown),
		Err:    errMsg,
	}
}

// RestHealth is the REST handler that reports the operational health of the service and
// its dependencies as a JSON object keyed by dependency.
type RestHealth struct{}

// NewRestHealth returns a new RestHealth handler.
func NewRestHealth() *RestHealth {
	return &RestHealth{}
}

func (handler *RestHealth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.Health")
	defer span.End()

	httpf.SendJSON(ctx, w, span, map[string]any{
		"client:postgres": NewRestHealthStatus(handler.reportPostgres(ctx)),
	})
}

func (handler *RestHealth) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(reportPostgres)")
	defer span.End()

	pg, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		// In transaction mode the pool is a transaction rather than a *bun.DB, so there is
		// no connection to ping; report healthy.
		return nil
	}

	err = pgdb.Ping()
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
