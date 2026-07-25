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
		result, err = handler.Handle(handlerCtx, request.Job, &jobExecuteProviderCallRecorder{
			jobsClient: service.jobsClient,
			job:        request.Job,
			workerID:   request.WorkerID,
		})

		cancelHandler()
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		job, settleErr := service.settleFailure(ctx, request, err)
		if settleErr != nil {
			return nil, otel.ReportError(span, settleErr)
		}

		span.SetAttributes(jobSpanAttributes(job)...)

		return job, nil
	}

	response, err := service.jobsClient.JobSettle(ctx, &servicejobs.JobSettleRequest{
		Id:       request.Job.GetId(),
		WorkerId: request.WorkerID,
		Outcome:  &servicejobs.JobSettleResult{Result: result},
	})
	if err != nil {
		settleErr := service.reportSettleError(ctx, request, err)

		return nil, otel.ReportError(span, settleErr)
	}

	span.SetAttributes(jobSpanAttributes(response.GetJob())...)

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

func (service *JobExecute) settleFailure(
	ctx context.Context,
	request *JobExecuteRequest,
	handlerErr error,
) (*servicejobs.Job, error) {
	failure := strconv.AppendQuote([]byte(`{"message":`), handlerErr.Error())
	failure = append(failure, '}')

	response, err := service.jobsClient.JobSettle(ctx, &servicejobs.JobSettleRequest{
		Id:       request.Job.GetId(),
		WorkerId: request.WorkerID,
		Outcome: &servicejobs.JobSettleFailure{Failure: &servicejobs.JobFailure{
			Error:     failure,
			Retryable: errors.Is(handlerErr, ErrJobRetryable),
		}},
	})
	if err != nil {
		return nil, service.reportSettleError(ctx, request, err)
	}

	return response.GetJob(), nil
}

func (service *JobExecute) reportSettleError(
	ctx context.Context,
	request *JobExecuteRequest,
	err error,
) error {
	if status.Code(err) == codes.FailedPrecondition {
		service.logger.Err(
			ctx,
			"job settle lost its claim",
			"job.id", request.Job.GetId(),
			"job.kind", request.Job.GetKind(),
			"job.attempt", request.Job.GetAttempt(),
			"job.status", request.Job.GetStatus().String(),
			"worker.id", request.WorkerID,
			"error", err,
		)
	}

	return fmt.Errorf("settle job: %w", err)
}

type jobExecuteProviderCallRecorder struct {
	jobsClient servicejobs.Client
	job        *servicejobs.Job
	workerID   string
}

func (recorder *jobExecuteProviderCallRecorder) Record(
	ctx context.Context, providerCallID string,
) error {
	_, err := recorder.jobsClient.JobRecordProviderCall(
		ctx,
		&servicejobs.JobRecordProviderCallRequest{
			Id:             recorder.job.GetId(),
			WorkerId:       recorder.workerID,
			ProviderCallId: providerCallID,
		},
	)
	if err != nil {
		return fmt.Errorf("record provider call: %w", err)
	}

	return nil
}
