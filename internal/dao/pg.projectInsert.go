package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectInsert.sql
var projectInsertQuery string

var ErrProjectInsertAlreadyExists = errors.New("project already exists")

type ProjectInsertRequest struct {
	ID       uuid.UUID
	Owner    uuid.UUID
	Lang     string
	Title    string
	Workflow []string
	Now      time.Time
}

type ProjectInsert struct{}

func NewProjectInsert() *ProjectInsert {
	return new(ProjectInsert)
}

func (repository *ProjectInsert) Exec(ctx context.Context, request *ProjectInsertRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ProjectInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("id", request.ID.String()),
		attribute.String("owner", request.Owner.String()),
		attribute.String("lang", request.Lang),
		attribute.String("title", request.Title),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Project)

	err = tx.NewRaw(
		projectInsertQuery,
		request.ID,
		request.Owner,
		request.Lang,
		request.Title,
		pgdialect.Array(request.Workflow),
		request.Now,
		request.Now,
	).Scan(ctx, entity)
	if err != nil {
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
			err = errors.Join(err, ErrProjectInsertAlreadyExists)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
