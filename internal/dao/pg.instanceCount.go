package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.instanceCount.sql
var instanceCountQuery string

// InstanceCountRequest is the input to [PgInstanceCount.Exec].
type InstanceCountRequest struct {
	ProjectID uuid.UUID
}

// InstanceCount is how many instances a project holds.
type InstanceCount struct {
	Count int `bun:"count"`
}

// PgInstanceCount counts the instances a project holds.
type PgInstanceCount struct{}

func NewPgInstanceCount() *PgInstanceCount {
	return &PgInstanceCount{}
}

func (dao *PgInstanceCount) Exec(ctx context.Context, request *InstanceCountRequest) (*InstanceCount, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgInstanceCount")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entity InstanceCount

	err = db.NewRaw(instanceCountQuery, request.ProjectID).Scan(ctx, &entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return &entity, nil
}
