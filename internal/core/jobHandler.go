package core

import (
	"context"
	"encoding/json"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"
)

// Job exposes the queue's work record inside the core package.
type Job = servicejobs.Job

// JobStatus identifies where a job sits in the queue lifecycle.
type JobStatus = servicejobs.JobStatus

const (
	// JobStatusUnspecified means the queue did not supply a lifecycle status.
	JobStatusUnspecified = servicejobs.JobStatusUnspecified
	// JobStatusPending means the job is waiting for a worker.
	JobStatusPending = servicejobs.JobStatusPending
	// JobStatusClaimed means a worker currently holds the job's lease.
	JobStatusClaimed = servicejobs.JobStatusClaimed
	// JobStatusSucceeded means the handler completed successfully.
	JobStatusSucceeded = servicejobs.JobStatusSucceeded
	// JobStatusFailed means the handler failed with no retry remaining.
	JobStatusFailed = servicejobs.JobStatusFailed
	// JobStatusAbandoned means recovery found an expired claim with no retry remaining.
	JobStatusAbandoned = servicejobs.JobStatusAbandoned
	// JobStatusCancelled means the owner stopped the job before completion.
	JobStatusCancelled = servicejobs.JobStatusCancelled
)

// A JobHandler performs the provider-neutral work described by a job.
//
// Handle must take the acting user from the job's OwnerID and ignore owner,
// actor, and user fields in Payload. Payload is caller-controlled JSON and is
// not an authority for side effects.
//
// Handle must not hold a database transaction across its work. A provider call
// can outlive a request and would pin a pooled connection and database snapshot.
//
// A resumable provider operation must record its identifier immediately after
// starting and before polling. On a later attempt, a non-empty ProviderCallID
// means Handle re-attaches to that operation instead of starting another one.
// Providers without resumable identifiers leave the recorder unused.
type JobHandler interface {
	// Kind returns the queue kind this handler accepts.
	Kind() string
	// Handle runs a job until it completes or its context is cancelled.
	Handle(ctx context.Context, job *Job, recorder ProviderCallRecorder) (json.RawMessage, error)
}

// ProviderCallRecorder makes a running provider operation recoverable after a
// worker crash.
type ProviderCallRecorder interface {
	// Record attaches the provider's operation identifier to the claimed job.
	Record(ctx context.Context, providerCallID string) error
}

// ErrJobRetryable marks a handler failure that the queue may requeue while an
// attempt remains. Handlers join it to the returned error.
var ErrJobRetryable = errors.New("job failed but may be retried")

func setJobSpanAttributes(span trace.Span, job *Job) {
	span.SetAttributes(
		attribute.String("job.id", job.GetId()),
		attribute.String("job.kind", job.GetKind()),
		attribute.Int("job.attempt", int(job.GetAttempt())),
		attribute.String("job.status", job.GetStatus().String()),
	)
}
