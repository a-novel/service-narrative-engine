package core

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

var errJobHandlerNotFound = errors.New("job handler not found")

// JobExecuteLogger records a lost claim fence that prevents settlement.
type JobExecuteLogger interface {
	// Err records a failed operation with request-scoped trace information.
	Err(ctx context.Context, message string, fields ...any)
}

// JobExecuteRequest carries a claimed job and the worker allowed to settle it.
type JobExecuteRequest struct {
	// Job is the claimed work to dispatch.
	Job *servicejobs.Job `validate:"required"`
	// WorkerID must match the worker holding the queue claim.
	WorkerID string `validate:"required,notblank"`
	// Deadline is the hard limit applied to the handler call.
	Deadline time.Duration `validate:"gt=0"`
}

// JobExecute dispatches claimed work and settles its result with the queue.
type JobExecute struct {
	jobsClient servicejobs.Client
	logger     JobExecuteLogger
	handlers   map[string]JobHandler
}

// NewJobExecute builds a dispatcher from the registered handler set. An empty
// set is valid and causes any accidentally claimed kind to settle as failed.
func NewJobExecute(
	jobsClient servicejobs.Client,
	logger JobExecuteLogger,
	handlers ...JobHandler,
) *JobExecute {
	handlersByKind := make(map[string]JobHandler, len(handlers))
	for _, handler := range handlers {
		handlersByKind[handler.Kind()] = handler
	}

	return &JobExecute{
		jobsClient: jobsClient,
		logger:     logger,
		handlers:   handlersByKind,
	}
}

// Exec runs a claimed job under its deadline and reports exactly one outcome
// to the queue. Handler failures are represented by the returned job state;
// errors report failures to validate or settle the run itself.
func (service *JobExecute) Exec(
	ctx context.Context,
	request *JobExecuteRequest,
) (*servicejobs.Job, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.JobExecute")
	defer span.End()

	if request != nil && request.Job != nil {
		span.SetAttributes(jobSpanAttributes(request.Job)...)
	}

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	handler, found := service.handlers[request.Job.GetKind()]

	var result []byte

	if !found {
		err = fmt.Errorf("%w: %q", errJobHandlerNotFound, request.Job.GetKind())
	} else {
		handlerCtx, cancelHandler := context.WithTimeout(ctx, request.Deadline)
		recorder := ProviderCallRecorder(func(providerCallID string) error {
			_, recordErr := service.jobsClient.JobRecordProviderCall(
				handlerCtx,
				&servicejobs.JobRecordProviderCallRequest{
					Id:             request.Job.GetId(),
					WorkerId:       request.WorkerID,
					ProviderCallId: providerCallID,
				},
			)
			if recordErr != nil {
				return fmt.Errorf("record provider call: %w", recordErr)
			}

			return nil
		})

		result, err = handler.Handle(handlerCtx, request.Job, recorder)

		cancelHandler()
	}

	settleRequest := &servicejobs.JobSettleRequest{
		Id:       request.Job.GetId(),
		WorkerId: request.WorkerID,
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		failure := strconv.AppendQuote([]byte(`{"message":`), err.Error())
		failure = append(failure, '}')
		settleRequest.Outcome = &servicejobs.JobSettleFailure{Failure: &servicejobs.JobFailure{
			Error:     failure,
			Retryable: errors.Is(err, ErrJobRetryable),
		}}
	} else {
		settleRequest.Outcome = &servicejobs.JobSettleResult{Result: result}
	}

	response, settleErr := service.jobsClient.JobSettle(ctx, settleRequest)
	if settleErr != nil {
		if status.Code(settleErr) == codes.FailedPrecondition {
			service.logger.Err(
				ctx,
				"job settle lost its claim",
				"job.id", request.Job.GetId(),
				"job.kind", request.Job.GetKind(),
				"job.attempt", request.Job.GetAttempt(),
				"job.status", request.Job.GetStatus().String(),
				"worker.id", request.WorkerID,
				"error", settleErr,
			)
		}

		return nil, otel.ReportError(span, fmt.Errorf("settle job: %w", settleErr))
	}

	span.SetAttributes(jobSpanAttributes(response.GetJob())...)

	if err != nil {
		return response.GetJob(), nil
	}

	return otel.ReportSuccess(span, response.GetJob()), nil
}

func jobSpanAttributes(job *servicejobs.Job) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("job.id", job.GetId()),
		attribute.String("job.kind", job.GetKind()),
		attribute.Int("job.attempt", int(job.GetAttempt())),
		attribute.String("job.status", job.GetStatus().String()),
	}
}
