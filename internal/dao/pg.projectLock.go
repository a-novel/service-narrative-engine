package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectLock.sql
var projectLockQuery string

// ErrProjectLockNotFound is returned by [PgProjectLock.Exec] when no project with the ID belongs to
// the owner.
var ErrProjectLockNotFound = errors.New("project not found")

// ProjectLockRequest is the input to [PgProjectLock.Exec].
type ProjectLockRequest struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
}

// PgProjectLock reads a project its owner holds and locks it until the transaction ends, so changes to
// the order of its scenes and layers run one at a time.
type PgProjectLock struct{}

func NewPgProjectLock() *PgProjectLock {
	return &PgProjectLock{}
}

func (dao *PgProjectLock) Exec(ctx context.Context, request *ProjectLockRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectLock")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Project{}

	err = db.NewRaw(projectLockQuery, request.ID, request.OwnerID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectLockNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
