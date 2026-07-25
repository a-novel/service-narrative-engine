package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"google.golang.org/grpc"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// JobEnqueueService is the queue operation JobEnqueue uses to submit work.
type JobEnqueueService interface {
	// JobEnqueue records a job or returns the existing idempotent submission.
	JobEnqueue(
		ctx context.Context, request *servicejobs.JobEnqueueRequest, opts ...grpc.CallOption,
	) (*servicejobs.JobEnqueueResponse, error)
}

// JobEnqueueRequest holds the trusted owner and opaque work submitted to the queue.
type JobEnqueueRequest struct {
	// Kind selects the handler that will run the job.
	Kind string `validate:"required,notblank,max=128"`
	// Payload is the handler's opaque JSON input.
	Payload json.RawMessage `validate:"required"`
	// Actor supplies the authenticated owner written to the queue.
	Actor Actor `validate:"required"`
	// IdempotencyKey re-attaches a repeated submission to the same job when set.
	IdempotencyKey string `validate:"omitempty,max=255"`
	// MaxAttempts caps runs of the job and defaults to one when zero.
	MaxAttempts int16 `validate:"min=1,max=32"`
}

// JobEnqueueResponse reports the recorded job and whether this call created it.
type JobEnqueueResponse struct {
	// Job is the created job or the existing idempotent submission.
	Job *Job
	// Created is false when the idempotency key matched an existing job.
	Created bool
}

// JobEnqueue validates and submits provider-neutral work to the queue.
type JobEnqueue struct {
	service JobEnqueueService
}

// NewJobEnqueue builds a JobEnqueue backed by the queue service.
func NewJobEnqueue(service JobEnqueueService) *JobEnqueue {
	return &JobEnqueue{service: service}
}

// Exec submits work with the authenticated actor as its owner.
func (service *JobEnqueue) Exec(
	ctx context.Context, request *JobEnqueueRequest,
) (*JobEnqueueResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.JobEnqueue")
	defer span.End()

	setJobSpanAttributes(span, &Job{
		Kind:   request.Kind,
		Status: JobStatusUnspecified,
	})

	resolvedRequest := *request
	if resolvedRequest.MaxAttempts == 0 {
		resolvedRequest.MaxAttempts = 1
	}

	err := validate.Struct(&resolvedRequest)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	serviceRequest := &servicejobs.JobEnqueueRequest{
		Kind:        resolvedRequest.Kind,
		Payload:     resolvedRequest.Payload,
		OwnerId:     resolvedRequest.Actor.UserID.String(),
		MaxAttempts: int32(resolvedRequest.MaxAttempts),
	}
	if resolvedRequest.IdempotencyKey != "" {
		serviceRequest.IdempotencyKey = &resolvedRequest.IdempotencyKey
	}

	response, err := service.service.JobEnqueue(ctx, serviceRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("enqueue job: %w", err))
	}

	setJobSpanAttributes(span, response.GetJob())

	return otel.ReportSuccess(span, &JobEnqueueResponse{
		Job:     response.GetJob(),
		Created: response.GetCreated(),
	}), nil
}
