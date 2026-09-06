package handlers

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/samber/lo"

	servicegenai "github.com/a-novel/service-genai/pkg/go"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

const restHealthCacheTTL = 30 * time.Second

var errRestHealthGenAIQueueMissing = errors.New("service-genai status omitted queue")

const (
	// RestHealthStatusUp marks a dependency the service can currently reach.
	RestHealthStatusUp = "up"
	// RestHealthStatusDown marks a dependency the service failed to reach.
	RestHealthStatusDown = "down"
)

// RestHealthStatus is the public health of one dependency.
type RestHealthStatus struct {
	// Status is either [RestHealthStatusUp] or [RestHealthStatusDown].
	Status string `json:"status"`
}

// RestHealthQueue describes the generation backlog without exposing request contents.
type RestHealthQueue struct {
	// Pending is the number of generation requests waiting to run.
	Pending int64 `json:"pending"`
	// OldestPendingAge is omitted when no generation is pending.
	OldestPendingAge string `json:"oldestPendingAge,omitempty"`
}

// RestHealthGenAIStatus adds queue depth to the service-genai dependency status.
type RestHealthGenAIStatus struct {
	RestHealthStatus

	Queue *RestHealthQueue `json:"queue,omitempty"`
}

// NewRestHealthStatus maps an error to a sanitized dependency status.
func NewRestHealthStatus(err error) *RestHealthStatus {
	return &RestHealthStatus{
		Status: lo.Ternary(err == nil, RestHealthStatusUp, RestHealthStatusDown),
	}
}

// RestHealth reports dependencies and caches the whole report to limit probe traffic.
type RestHealth struct {
	apiJSONKeys servicejsonkeys.BaseClient
	apiGenAI    servicegenai.Client

	cacheMutex     sync.Mutex
	cachedReport   map[string]any
	cacheExpiresAt time.Time
}

// NewRestHealth returns a health handler backed by the service clients.
func NewRestHealth(apiJSONKeys servicejsonkeys.BaseClient, apiGenAI servicegenai.Client) *RestHealth {
	return &RestHealth{apiJSONKeys: apiJSONKeys, apiGenAI: apiGenAI}
}

func (handler *RestHealth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.Health")
	defer span.End()

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, handler.report(ctx))
}

func (handler *RestHealth) report(ctx context.Context) map[string]any {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(report)")
	defer span.End()

	handler.cacheMutex.Lock()
	defer handler.cacheMutex.Unlock()

	now := time.Now()
	if handler.cachedReport != nil && now.Before(handler.cacheExpiresAt) {
		return otel.ReportSuccess(span, handler.cachedReport)
	}

	genaiQueue, genaiErr := handler.reportGenAI(ctx)

	handler.cachedReport = map[string]any{
		"client:postgres":  NewRestHealthStatus(handler.reportPostgres(ctx)),
		"client:json-keys": NewRestHealthStatus(handler.reportJSONKeys(ctx)),
		"client:genai": &RestHealthGenAIStatus{
			RestHealthStatus: *NewRestHealthStatus(genaiErr),
			Queue:            genaiQueue,
		},
	}
	handler.cacheExpiresAt = now.Add(restHealthCacheTTL)

	return otel.ReportSuccess(span, handler.cachedReport)
}

func (handler *RestHealth) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(reportPostgres)")
	defer span.End()

	err := postgres.Health(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

func (handler *RestHealth) reportJSONKeys(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(reportJSONKeys)")
	defer span.End()

	_, err := handler.apiJSONKeys.Status(ctx, &servicejsonkeys.StatusRequest{})
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

func (handler *RestHealth) reportGenAI(ctx context.Context) (*RestHealthQueue, error) {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(reportGenAI)")
	defer span.End()

	response, err := handler.apiGenAI.Status(ctx, &servicegenai.StatusRequest{})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	queue := response.GetQueue()
	if queue == nil {
		return nil, otel.ReportError(span, errRestHealthGenAIQueueMissing)
	}

	report := &RestHealthQueue{Pending: queue.GetPending()}
	if queue.GetPending() > 0 {
		report.OldestPendingAge = time.Duration(
			queue.GetOldestPendingAgeSeconds() * float64(time.Second),
		).String()
	}

	return otel.ReportSuccess(span, report), nil
}
