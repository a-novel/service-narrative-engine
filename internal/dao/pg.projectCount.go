package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectCount.sql
var projectCountQuery string

// ProjectCountRequest is the input to [PgProjectCount.Exec].
type ProjectCountRequest struct {
	OwnerID uuid.UUID
}

// ProjectCount is how many projects a user owns.
type ProjectCount struct {
	Count int `bun:"count"`
}

// PgProjectCount counts the projects a user owns.
type PgProjectCount struct{}

func NewPgProjectCount() *PgProjectCount {
	return &PgProjectCount{}
}

func (dao *PgProjectCount) Exec(ctx context.Context, request *ProjectCountRequest) (*ProjectCount, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectCount")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entity ProjectCount

	err = db.NewRaw(projectCountQuery, request.OwnerID).Scan(ctx, &entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return &entity, nil
}
