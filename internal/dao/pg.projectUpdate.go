package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectUpdate.sql
var projectUpdateQuery string

var ErrProjectUpdateNotFound = errors.New("project not found")

type ProjectUpdateRequest struct {
	ID       uuid.UUID
	Title    string
	Workflow []string
	Now      time.Time
}

type ProjectUpdate struct{}

func NewProjectUpdate() *ProjectUpdate {
	return new(ProjectUpdate)
}

func (repository *ProjectUpdate) Exec(ctx context.Context, request *ProjectUpdateRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ProjectUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("id", request.ID.String()),
		attribute.String("title", request.Title),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Project)

	err = tx.NewRaw(
		projectUpdateQuery,
		request.ID,
		request.Title,
		pgdialect.Array(request.Workflow),
		request.Now,
	).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrProjectUpdateNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
