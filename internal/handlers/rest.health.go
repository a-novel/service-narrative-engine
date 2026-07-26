package handlers

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/uptrace/bun"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

const restHealthCacheTTL = 30 * time.Second

var errRestHealthJobsQueueMissing = errors.New("service-jobs status omitted queue")

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

// RestHealthQueue describes the jobs backlog without exposing job contents.
type RestHealthQueue struct {
	// Pending is the number of due jobs still waiting for a worker.
	Pending int64 `json:"pending"`
	// OldestPendingAge is omitted when no job is pending.
	OldestPendingAge string `json:"oldestPendingAge,omitempty"`
}

// RestHealthJobsStatus adds queue depth to the service-jobs dependency status.
type RestHealthJobsStatus struct {
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
	jobs        servicejobs.Client

	cacheMutex     sync.Mutex
	cachedReport   map[string]any
	cacheExpiresAt time.Time
}

// NewRestHealth returns a health handler backed by the service clients.
func NewRestHealth(apiJSONKeys servicejsonkeys.BaseClient, jobs servicejobs.Client) *RestHealth {
	return &RestHealth{apiJSONKeys: apiJSONKeys, jobs: jobs}
}

func (handler *RestHealth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.Health")
	defer span.End()

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, handler.report(ctx))
}

func (handler *RestHealth) report(ctx context.Context) map[string]any {
	handler.cacheMutex.Lock()
	defer handler.cacheMutex.Unlock()

	now := time.Now()
	if handler.cachedReport != nil && now.Before(handler.cacheExpiresAt) {
		return handler.cachedReport
	}

	jobsQueue, jobsErr := handler.reportJobs(ctx)

	handler.cachedReport = map[string]any{
		"client:postgres":  NewRestHealthStatus(handler.reportPostgres(ctx)),
		"client:json-keys": NewRestHealthStatus(handler.reportJSONKeys(ctx)),
		"client:jobs": &RestHealthJobsStatus{
			RestHealthStatus: *NewRestHealthStatus(jobsErr),
			Queue:            jobsQueue,
		},
	}
	handler.cacheExpiresAt = now.Add(restHealthCacheTTL)

	return handler.cachedReport
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
		// In transaction mode the pooled handle is a transaction, which exposes
		// no Ping; treat the dependency as healthy.
		return nil
	}

	err = pgdb.Ping()
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

func (handler *RestHealth) reportJobs(ctx context.Context) (*RestHealthQueue, error) {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(reportJobs)")
	defer span.End()

	response, err := handler.jobs.Status(ctx, &servicejobs.StatusRequest{})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	queue := response.GetQueue()
	if queue == nil {
		return nil, otel.ReportError(span, errRestHealthJobsQueueMissing)
	}

	report := &RestHealthQueue{Pending: queue.GetPending()}
	if age := queue.GetOldestPendingAge(); age != nil {
		report.OldestPendingAge = age.AsDuration().String()
	}

	return otel.ReportSuccess(span, report), nil
}
