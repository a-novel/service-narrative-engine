package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sceneLock.sql
var sceneLockQuery string

// SceneLockRequest is the input to [PgSceneLock.Exec].
type SceneLockRequest struct {
	ProjectID uuid.UUID
	// IDs lists the scenes to lock.
	IDs []uuid.UUID
}

// PgSceneLock locks scenes of a project until the transaction ends, so writes to their cells run one at a
// time. It skips an ID the project does not hold.
type PgSceneLock struct{}

func NewPgSceneLock() *PgSceneLock {
	return &PgSceneLock{}
}

func (dao *PgSceneLock) Exec(ctx context.Context, request *SceneLockRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSceneLock")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	_, err = db.NewRaw(sceneLockQuery, request.ProjectID, bun.List(request.IDs), len(request.IDs)).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
