package dao

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// JobStatus is where a job sits in its lifecycle. The database holds it as text with a CHECK list
// rather than an enum, so adding a value stays a plain migration.
type JobStatus string

const (
	// JobStatusPending marks a job waiting to be claimed. It is the state every job is enqueued in.
	JobStatusPending JobStatus = "pending"
	// JobStatusClaimed marks a job a worker holds a lease on and is currently running.
	JobStatusClaimed JobStatus = "claimed"
	// JobStatusSucceeded marks a job whose handler returned a result.
	JobStatusSucceeded JobStatus = "succeeded"
	// JobStatusFailed marks a job whose handler returned an error it had no attempts left to retry.
	JobStatusFailed JobStatus = "failed"
	// JobStatusAbandoned marks a job the reaper settled because its lease expired and no attempt
	// remained — the worker running it died or was killed mid-run.
	JobStatusAbandoned JobStatus = "abandoned"
	// JobStatusCancelled marks a job settled on the owner's request rather than by running it.
	JobStatusCancelled JobStatus = "cancelled"
)

// A Job is one unit of asynchronous work, durable across restarts. It carries an opaque Kind and
// Payload: nothing in this table knows what the work is.
//
// Two invariants are enforced by the database rather than by convention. A lease exists exactly when
// the status is claimed, and the settled and expiry timestamps are set exactly when the status is
// terminal. A partial write that satisfies one and not the other is rejected.
type Job struct {
	bun.BaseModel `bun:"table:jobs,alias:jobs"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// Kind selects the handler that runs this job. Never empty.
	Kind string `bun:"kind"`
	// Payload is the caller-supplied input, opaque to the queue. A handler reads its owner from
	// OwnerID and never from here: the payload is caller-supplied, so an owner sourced from it would
	// let a caller attribute their job's side effects to someone else.
	Payload json.RawMessage `bun:"payload,type:jsonb"`

	// OwnerID is the user the job acts on behalf of, taken from the verified access token. Reads are
	// scoped by it inside the statement, so this column is the whole ownership guarantee.
	OwnerID uuid.UUID `bun:"owner_id,type:uuid"`
	// IdempotencyKey deduplicates repeat submissions, scoped to one owner and one kind. Nil for job
	// kinds cheap enough to run twice.
	IdempotencyKey *string `bun:"idempotency_key"`
	// RequestFingerprint is the digest of the request that enqueued this job, used to tell a genuine
	// replay from the same key submitted with a different body.
	RequestFingerprint []byte `bun:"request_fingerprint"`

	Status JobStatus `bun:"status"`

	// Attempt counts runs already started.
	Attempt int16 `bun:"attempt"`
	// MaxAttempts caps them. One by default, which is the right floor for a priced call.
	MaxAttempts int16 `bun:"max_attempts"`

	// RunAt is the earliest time a worker may claim this job.
	RunAt time.Time `bun:"run_at"`
	// LeaseExpiresAt is when the current claim lapses and the reaper may recover the job. Set exactly
	// while the status is claimed.
	LeaseExpiresAt *time.Time `bun:"lease_expires_at"`
	// ClaimedBy identifies the worker holding the lease.
	ClaimedBy *string `bun:"claimed_by"`
	// CancelRequestedAt records an owner's cancellation request. Nothing writes it yet.
	CancelRequestedAt *time.Time `bun:"cancel_requested_at"`

	// ProviderCallID is the third party's identifier for the operation this job started. It is what
	// lets a reclaimed job re-attach to that operation instead of paying for it twice.
	ProviderCallID *string `bun:"provider_call_id"`

	// Result holds the handler's output on success.
	Result json.RawMessage `bun:"result,type:jsonb"`
	// Error holds the structured failure on a terminal failure.
	Error json.RawMessage `bun:"error,type:jsonb"`

	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
	// SettledAt is when the job reached a terminal status. Set exactly while it is terminal.
	SettledAt *time.Time `bun:"settled_at"`
	// ExpiresAt is when the purger may delete the row. Set exactly while the job is terminal.
	ExpiresAt *time.Time `bun:"expires_at"`
}
