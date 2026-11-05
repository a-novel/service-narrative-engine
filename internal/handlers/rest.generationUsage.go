package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestGenerationUsageService is the core operation [RestGenerationUsage] delegates to.
type RestGenerationUsageService interface {
	Exec(ctx context.Context, request *core.GenerationUsageRequest) (*core.GenerationUsageSummary, error)
}

// RestGenerationUsageSummary is the caller's caps, and what the caller consumed under each.
type RestGenerationUsageSummary struct {
	// Caps are the caps genai counts the caller's generations against. Empty caps limit nothing.
	Caps *RestGenerationCaps `json:"caps"`
	// InFlight is how many of the caller's calls count against the concurrency.
	InFlight int64 `json:"inFlight"`
	// Windows holds the tokens the caller's calls consumed over each window of the caps, in their order.
	Windows []*RestGenerationUsageWindow `json:"windows"`
}

// RestGenerationCaps limit a caller's generations. A limit left out does not apply.
type RestGenerationCaps struct {
	// Concurrency caps the calls the caller may have in flight at once.
	Concurrency *int32 `json:"concurrency,omitempty"`
	// Windows cap the tokens the caller's calls consume over rolling windows, shortest first.
	Windows []*RestGenerationCapWindow `json:"windows,omitempty"`
}

// RestGenerationCapWindow caps the tokens a caller's calls consume over one rolling window.
type RestGenerationCapWindow struct {
	// DurationSeconds is how far back the window reaches, in seconds.
	DurationSeconds int64 `json:"durationSeconds"`

	// InputTokens caps the input tokens, cached ones included.
	InputTokens *int64 `json:"inputTokens,omitempty"`
	// OutputTokens caps the output tokens, reasoning ones included.
	OutputTokens *int64 `json:"outputTokens,omitempty"`
}

// RestGenerationUsageWindow is what a caller's calls consumed over one rolling window.
type RestGenerationUsageWindow struct {
	// DurationSeconds is how far back the window reaches, in seconds.
	DurationSeconds int64 `json:"durationSeconds"`

	// InputTokens is the input tokens consumed in the window, cached ones included.
	InputTokens int64 `json:"inputTokens"`
	// OutputTokens is the output tokens consumed in the window, reasoning ones included.
	OutputTokens int64 `json:"outputTokens"`
}

// RestGenerationUsage is the HTTP handler for the generation-usage endpoint. It responds with the caps of the
// caller's roles and the caller's usage under each, so a client shows the room left before a submit waits.
type RestGenerationUsage struct {
	service RestGenerationUsageService
	logger  logging.Log
}

func NewRestGenerationUsage(service RestGenerationUsageService, logger logging.Log) *RestGenerationUsage {
	return &RestGenerationUsage{service: service, logger: logger}
}

func (handler *RestGenerationUsage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.GenerationUsage")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	summary, err := handler.service.Exec(ctx, &core.GenerationUsageRequest{OwnerID: ownerID, Roles: requestRoles(r)})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, generationErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadGenerationUsage(summary))
}

func loadGenerationUsage(summary *core.GenerationUsageSummary) *RestGenerationUsageSummary {
	return &RestGenerationUsageSummary{
		Caps: &RestGenerationCaps{
			Concurrency: summary.Caps.Concurrency,
			Windows: lo.Map(summary.Caps.Windows, func(window *core.GenerationCapWindow, _ int) *RestGenerationCapWindow {
				return &RestGenerationCapWindow{
					DurationSeconds: int64(window.Duration / time.Second),
					InputTokens:     window.InputTokens,
					OutputTokens:    window.OutputTokens,
				}
			}),
		},
		InFlight: summary.InFlight,
		Windows: lo.Map(summary.Windows, func(window *core.GenerationUsageWindow, _ int) *RestGenerationUsageWindow {
			return &RestGenerationUsageWindow{
				DurationSeconds: int64(window.Duration / time.Second),
				InputTokens:     window.InputTokens,
				OutputTokens:    window.OutputTokens,
			}
		}),
	}
}
