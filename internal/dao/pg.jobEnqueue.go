package dao

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.jobEnqueue.sql
var jobEnqueueQuery string

// JobEnqueueRequest holds the parameters for a [JobEnqueue.Exec] call.
type JobEnqueueRequest struct {
	// ID is minted by the caller rather than the database, which is what makes the winner of an
	// idempotency race identifiable: the returned row carries the winner's id, so a caller compares
	// it against the one it generated. It must be a v7 UUID, matching the column's own default.
	//
	// It is never taken from the client. The public request carries no id field, so a caller cannot
	// choose one and collide with a row it does not own.
	ID uuid.UUID
	// Kind selects the handler. Rejected by the database when empty.
	Kind string
	// Payload is the handler's opaque input. It must be valid JSON; the column is NOT NULL, so a nil
	// value fails the insert rather than storing one.
	Payload json.RawMessage
	// OwnerID is the acting user from the verified access token, never a value from the payload.
	OwnerID uuid.UUID
	// IdempotencyKey deduplicates within one owner and kind. Nil skips deduplication entirely — the
	// partial unique index does not cover null keys, so every such call inserts a new row.
	IdempotencyKey *string
	// RequestFingerprint is the digest of the request body, which a later call under the same key is
	// compared against. Callers pass the digest of an empty body rather than nothing, so the stored
	// side is never null.
	RequestFingerprint []byte
	// MaxAttempts caps the runs this job gets. One is the right value for a priced call.
	MaxAttempts int16
}

// A JobEnqueue records a unit of asynchronous work, or returns the one already recorded under the
// same owner, kind and idempotency key.
//
// It always returns a row. The loser of a race gets the winner's job rather than an error, so a
// caller can attach to work already in flight; comparing the returned ID against the requested one
// is what tells the two apart.
//
// The database handle comes from the context, so an enqueue made inside a caller's transaction is
// part of that caller's unit of work.
//
// A conflicting insert takes a row lock, so two concurrent calls under one key serialize until the
// first commits. Enqueue exactly one key per transaction: inserting several in a caller-determined
// order is what would turn that lock into a deadlock between two callers holding each other's rows.
type JobEnqueue struct{}

// NewJobEnqueue returns a new JobEnqueue DAO.
func NewJobEnqueue() *JobEnqueue {
	return new(JobEnqueue)
}

func (dao *JobEnqueue) Exec(ctx context.Context, request *JobEnqueueRequest) (*Job, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.JobEnqueue")
	defer span.End()

	span.SetAttributes(
		attribute.String("job.id", request.ID.String()),
		attribute.String("job.kind", request.Kind),
		attribute.String("job.owner_id", request.OwnerID.String()),
		attribute.Bool("job.idempotent", request.IdempotencyKey != nil),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Job)

	err = tx.NewRaw(
		jobEnqueueQuery,
		request.ID,
		request.Kind,
		request.Payload,
		request.OwnerID,
		request.IdempotencyKey,
		request.RequestFingerprint,
		request.MaxAttempts,
	).Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
