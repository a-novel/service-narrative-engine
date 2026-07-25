package core

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

var errJobHandlerNotFound = errors.New("job handler not found")

// JobExecuteServiceSettle is the queue operation JobExecute uses to report an outcome.
type JobExecuteServiceSettle interface {
	// JobSettle transitions the claimed job from the worker's reported outcome.
	JobSettle(
		ctx context.Context, request *servicejobs.JobSettleRequest, opts ...grpc.CallOption,
	) (*servicejobs.JobSettleResponse, error)
}

// JobExecuteServiceRecordProviderCall is the queue operation that makes a
// provider operation recoverable by a later attempt.
type JobExecuteServiceRecordProviderCall interface {
	// JobRecordProviderCall attaches a provider operation to the claimed job.
	JobRecordProviderCall(
		ctx context.Context,
		request *servicejobs.JobRecordProviderCallRequest,
		opts ...grpc.CallOption,
	) (*servicejobs.JobRecordProviderCallResponse, error)
}

// JobExecuteLogger records a lost claim fence that prevents settlement.
type JobExecuteLogger interface {
	// Err records a failed operation with request-scoped trace information.
	Err(ctx context.Context, message string, fields ...any)
}

// JobExecuteRequest carries a claimed job and the worker allowed to settle it.
type JobExecuteRequest struct {
	// Job is the claimed work to dispatch.
	Job *Job `validate:"required"`
	// WorkerID must match the worker holding the queue claim.
	WorkerID string `validate:"required,notblank"`
	// Deadline is the hard limit applied to the handler call.
	Deadline time.Duration `validate:"gt=0"`
}

// JobExecute dispatches claimed work and settles its result with the queue.
type JobExecute struct {
	jobSettle             JobExecuteServiceSettle
	jobRecordProviderCall JobExecuteServiceRecordProviderCall
	logger                JobExecuteLogger
	handlers              map[string]JobHandler
}

// NewJobExecute builds a dispatcher from the registered handler set. An empty
// set is valid and causes any accidentally claimed kind to settle as failed.
func NewJobExecute(
	jobSettle JobExecuteServiceSettle,
	jobRecordProviderCall JobExecuteServiceRecordProviderCall,
	logger JobExecuteLogger,
	handlers ...JobHandler,
) *JobExecute {
	handlersByKind := make(map[string]JobHandler, len(handlers))
	for _, handler := range handlers {
		handlersByKind[handler.Kind()] = handler
	}

	return &JobExecute{
		jobSettle:             jobSettle,
		jobRecordProviderCall: jobRecordProviderCall,
		logger:                logger,
		handlers:              handlersByKind,
	}
}

// Exec runs a claimed job under its deadline and reports exactly one outcome
// to the queue. Handler failures are represented by the returned job state;
// errors report failures to validate or settle the run itself.
func (service *JobExecute) Exec(ctx context.Context, request *JobExecuteRequest) (*Job, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.JobExecute")
	defer span.End()

	setJobSpanAttributes(span, &Job{Status: JobStatusUnspecified})

	if request != nil && request.Job != nil {
		setJobSpanAttributes(span, request.Job)
	}

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	handler, found := service.handlers[request.Job.GetKind()]
	if !found {
		err = fmt.Errorf("%w: %q", errJobHandlerNotFound, request.Job.GetKind())
		_ = otel.ReportError(span, err)

		return service.settleFailure(ctx, span, request, err)
	}

	handlerCtx, cancelHandler := context.WithTimeout(ctx, request.Deadline)
	result, err := handler.Handle(handlerCtx, request.Job, &jobExecuteProviderCallRecorder{
		service:  service.jobRecordProviderCall,
		job:      request.Job,
		workerID: request.WorkerID,
	})

	cancelHandler()

	if err != nil {
		_ = otel.ReportError(span, err)

		return service.settleFailure(ctx, span, request, err)
	}

	response, err := service.jobSettle.JobSettle(ctx, &servicejobs.JobSettleRequest{
		Id:       request.Job.GetId(),
		WorkerId: request.WorkerID,
		Outcome:  &servicejobs.JobSettleResult{Result: result},
	})
	if err != nil {
		settleErr := service.reportSettleError(ctx, request, err)

		return nil, otel.ReportError(span, settleErr)
	}

	setJobSpanAttributes(span, response.GetJob())

	return otel.ReportSuccess(span, response.GetJob()), nil
}

func (service *JobExecute) settleFailure(
	ctx context.Context,
	span trace.Span,
	request *JobExecuteRequest,
	handlerErr error,
) (*Job, error) {
	ctx, settleSpan := otel.Tracer().Start(ctx, "core.JobExecute(settleFailure)")
	defer settleSpan.End()

	setJobSpanAttributes(settleSpan, request.Job)

	failure := strconv.AppendQuote([]byte(`{"message":`), handlerErr.Error())
	failure = append(failure, '}')

	response, err := service.jobSettle.JobSettle(ctx, &servicejobs.JobSettleRequest{
		Id:       request.Job.GetId(),
		WorkerId: request.WorkerID,
		Outcome: &servicejobs.JobSettleFailure{Failure: &servicejobs.JobFailure{
			Error:     failure,
			Retryable: errors.Is(handlerErr, ErrJobRetryable),
		}},
	})
	if err != nil {
		settleErr := service.reportSettleError(ctx, request, err)
		_ = otel.ReportError(span, settleErr)

		return nil, otel.ReportError(settleSpan, settleErr)
	}

	setJobSpanAttributes(span, response.GetJob())
	setJobSpanAttributes(settleSpan, response.GetJob())

	return otel.ReportSuccess(settleSpan, response.GetJob()), nil
}

func (service *JobExecute) reportSettleError(
	ctx context.Context,
	request *JobExecuteRequest,
	err error,
) error {
	ctx, span := otel.Tracer().Start(ctx, "core.JobExecute(reportSettleError)")
	defer span.End()

	setJobSpanAttributes(span, request.Job)

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

	return otel.ReportError(span, fmt.Errorf("settle job: %w", err))
}

type jobExecuteProviderCallRecorder struct {
	service  JobExecuteServiceRecordProviderCall
	job      *Job
	workerID string
}

func (recorder *jobExecuteProviderCallRecorder) Record(
	ctx context.Context, providerCallID string,
) error {
	ctx, span := otel.Tracer().Start(ctx, "core.JobExecute(recordProviderCall)")
	defer span.End()

	setJobSpanAttributes(span, recorder.job)

	_, err := recorder.service.JobRecordProviderCall(ctx, &servicejobs.JobRecordProviderCallRequest{
		Id:             recorder.job.GetId(),
		WorkerId:       recorder.workerID,
		ProviderCallId: providerCallID,
	})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("record provider call: %w", err))
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
