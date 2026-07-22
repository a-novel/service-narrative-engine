package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.jobGet.sql
var jobGetQuery string

// ErrJobGetNotFound is returned when no job matches both the requested ID and owner. A job owned by
// someone else is reported the same way as one that does not exist, so the caller cannot tell them
// apart; handlers map this to 404 and never to 403.
var ErrJobGetNotFound = errors.New("job not found")

// JobGetRequest holds the parameters for a [JobGet.Exec] call.
type JobGetRequest struct {
	ID uuid.UUID
	// OwnerID scopes the read. It is part of the query predicate, so omitting it returns no rows
	// rather than another owner's job.
	OwnerID uuid.UUID
}

// A JobGet retrieves one of an owner's jobs by its ID.
type JobGet struct{}

// NewJobGet returns a new JobGet DAO.
func NewJobGet() *JobGet {
	return new(JobGet)
}

func (dao *JobGet) Exec(ctx context.Context, request *JobGetRequest) (*Job, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.JobGet")
	defer span.End()

	span.SetAttributes(
		attribute.String("job.id", request.ID.String()),
		attribute.String("job.owner_id", request.OwnerID.String()),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Job)

	err = tx.NewRaw(jobGetQuery, request.ID, request.OwnerID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrJobGetNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
