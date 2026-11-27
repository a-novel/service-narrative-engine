package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.instanceList.sql
var instanceListQuery string

// InstanceListRequest is the input to [PgInstanceList.Exec].
type InstanceListRequest struct {
	ProjectID uuid.UUID

	// Limit is the most instances to return.
	Limit int
}

// PgInstanceList lists the instances of a project, oldest first, without their content.
type PgInstanceList struct{}

func NewPgInstanceList() *PgInstanceList {
	return &PgInstanceList{}
}

func (dao *PgInstanceList) Exec(ctx context.Context, request *InstanceListRequest) ([]*InstanceDescriptor, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgInstanceList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entities := []*InstanceDescriptor{}

	err = db.NewRaw(instanceListQuery, request.ProjectID, request.Limit).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entities, nil
}
