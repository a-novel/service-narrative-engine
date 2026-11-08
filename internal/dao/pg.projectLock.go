package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectLock.sql
var projectLockQuery string

// ProjectLockRequest is the input to [PgProjectLock.Exec].
type ProjectLockRequest struct {
	ID uuid.UUID
	// OwnerID is the user who claims the project.
	OwnerID uuid.UUID

	// Now is the instant stored as edited_at.
	Now time.Time
}

// PgProjectLock locks a project its owner names until the transaction ends, so the writes to it run one at
// a time, and stamps its edited_at. It reports whether the user owns the project, and locks and stamps
// nothing when they do not. A missing project yields false with a nil error.
type PgProjectLock struct{}

func NewPgProjectLock() *PgProjectLock {
	return &PgProjectLock{}
}

func (dao *PgProjectLock) Exec(ctx context.Context, request *ProjectLockRequest) (bool, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectLock")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	res, err := db.NewRaw(projectLockQuery, request.ID, request.OwnerID, request.Now).Exec(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	return n > 0, nil
}
