package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.instanceLinksList.sql
var instanceLinksListQuery string

// InstanceLinksListRequest is the input to [PgInstanceLinksList.Exec].
type InstanceLinksListRequest struct {
	ProjectID uuid.UUID

	// Limit caps the instances in the page.
	Limit int
	// Offset is how many instances come before the page, oldest first.
	Offset int
}

// InstanceLinksListRow is the links of one instance of a page.
type InstanceLinksListRow struct {
	InstanceLinks

	// Total counts every instance of the project, beyond the page.
	Total int `bun:"total"`
}

// PgInstanceLinksList reads a page of the links the instances of a project store, oldest instance first. An
// instance that never wrote its links comes with an empty list. Each row carries the total, and a page past the
// end comes back empty.
type PgInstanceLinksList struct{}

func NewPgInstanceLinksList() *PgInstanceLinksList {
	return &PgInstanceLinksList{}
}

func (dao *PgInstanceLinksList) Exec(
	ctx context.Context, request *InstanceLinksListRequest,
) ([]*InstanceLinksListRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgInstanceLinksList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entities := []*InstanceLinksListRow{}

	err = db.NewRaw(instanceLinksListQuery, request.ProjectID, request.Limit, request.Offset).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entities, nil
}
