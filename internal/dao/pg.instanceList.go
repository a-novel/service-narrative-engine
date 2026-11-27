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

	// Limit caps the instances in the page.
	Limit int
	// Offset is how many instances come before the page, oldest first.
	Offset int
}

// InstanceListRow is one instance of a page, without its content.
type InstanceListRow struct {
	InstanceDescriptor

	// Total counts every instance of the project, beyond the page.
	Total int `bun:"total"`
}

// PgInstanceList reads a page of the instances of a project, oldest first, without their content. Each row
// carries the total, and a page past the end comes back empty.
type PgInstanceList struct{}

func NewPgInstanceList() *PgInstanceList {
	return &PgInstanceList{}
}

func (dao *PgInstanceList) Exec(ctx context.Context, request *InstanceListRequest) ([]*InstanceListRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgInstanceList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entities := []*InstanceListRow{}

	err = db.NewRaw(instanceListQuery, request.ProjectID, request.Limit, request.Offset).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entities, nil
}
