package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// ErrJobGetNotFound is returned when no job matches both the requested ID and
// authenticated owner.
var ErrJobGetNotFound = errors.New("job not found")

// JobGetService is the owner-scoped queue operation JobGet uses to read work.
type JobGetService interface {
	// JobGet returns a job only when the supplied owner owns it.
	JobGet(
		ctx context.Context, request *servicejobs.JobGetRequest, opts ...grpc.CallOption,
	) (*servicejobs.JobGetResponse, error)
}

// JobGetRequest identifies an owned job to read.
type JobGetRequest struct {
	// ID identifies the job.
	ID uuid.UUID `validate:"required"`
	// Actor supplies the authenticated owner used to scope the lookup.
	Actor Actor `validate:"required"`
}

// JobGet reads one job without revealing jobs owned by another actor.
type JobGet struct {
	service JobGetService
}

// NewJobGet builds a JobGet backed by the queue service.
func NewJobGet(service JobGetService) *JobGet {
	return &JobGet{service: service}
}

// Exec returns the job when it belongs to the authenticated actor.
func (service *JobGet) Exec(ctx context.Context, request *JobGetRequest) (*Job, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.JobGet")
	defer span.End()

	setJobSpanAttributes(span, &Job{
		Id:     request.ID.String(),
		Status: JobStatusUnspecified,
	})

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	response, err := service.service.JobGet(ctx, &servicejobs.JobGetRequest{
		Id:      request.ID.String(),
		OwnerId: request.Actor.UserID.String(),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			err = errors.Join(err, ErrJobGetNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("get job: %w", err))
	}

	setJobSpanAttributes(span, response.GetJob())

	return otel.ReportSuccess(span, response.GetJob()), nil
}
