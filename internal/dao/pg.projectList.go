package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectList.sql
var projectListQuery string

type ProjectListRequest struct {
	Owner  uuid.UUID
	Limit  int
	Offset int
}

type ProjectList struct{}

func NewProjectList() *ProjectList {
	return new(ProjectList)
}

func (repository *ProjectList) Exec(ctx context.Context, request *ProjectListRequest) ([]*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ProjectList")
	defer span.End()

	span.SetAttributes(
		attribute.String("owner", request.Owner.String()),
		attribute.Int("data.limit", request.Limit),
		attribute.Int("data.offset", request.Offset),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var projects []*Project

	err = tx.NewRaw(
		projectListQuery,
		request.Owner,
		bun.NullZero(request.Limit),
		request.Offset,
	).Scan(ctx, &projects)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	if projects == nil {
		projects = []*Project{}
	}

	return otel.ReportSuccess(span, projects), nil
}
