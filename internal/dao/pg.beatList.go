package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.beatList.sql
var beatListQuery string

// BeatListRequest is the input to [PgBeatList.Exec].
type BeatListRequest struct {
	// ProjectID is the project whose beats to read.
	ProjectID uuid.UUID

	// Limit caps the beats in the page.
	Limit int
	// Offset is how many beats in narrative order come before the page.
	Offset int
}

// BeatListRow is one beat of a page.
type BeatListRow struct {
	Beat

	// Total counts every beat of the project, beyond the page.
	Total int `bun:"total"`
}

// PgBeatList reads a page of a project's beats, in narrative order. Each row carries the total,
// and a page past the end comes back empty.
type PgBeatList struct{}

func NewPgBeatList() *PgBeatList {
	return &PgBeatList{}
}

func (dao *PgBeatList) Exec(ctx context.Context, request *BeatListRequest) ([]*BeatListRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgBeatList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*BeatListRow

	err = db.NewRaw(beatListQuery, request.ProjectID, request.Limit, request.Offset).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("beats.count", len(entities)))

	return entities, nil
}
