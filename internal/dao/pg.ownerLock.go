package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.ownerLock.sql
var ownerLockQuery string

// OwnerLockRequest is the input to [PgOwnerLock.Exec].
type OwnerLockRequest struct {
	// OwnerID is the user whose project creations to serialize.
	OwnerID uuid.UUID
}

// PgOwnerLock serializes the project creations of one user until the transaction ends, so a count of the
// user's projects read after it stays true until the insert commits. It takes a transaction-scoped
// advisory lock, since no row stands for the user in this service.
type PgOwnerLock struct{}

func NewPgOwnerLock() *PgOwnerLock {
	return &PgOwnerLock{}
}

func (dao *PgOwnerLock) Exec(ctx context.Context, request *OwnerLockRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgOwnerLock")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	_, err = db.NewRaw(ownerLockQuery, request.OwnerID).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
