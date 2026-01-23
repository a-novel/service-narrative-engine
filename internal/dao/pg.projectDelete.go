package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectDelete.sql
var projectDeleteQuery string

var ErrProjectDeleteNotFound = errors.New("project not found")

type ProjectDeleteRequest struct {
	ID uuid.UUID
}

type ProjectDelete struct{}

func NewProjectDelete() *ProjectDelete {
	return new(ProjectDelete)
}

func (repository *ProjectDelete) Exec(ctx context.Context, request *ProjectDeleteRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ProjectDelete")
	defer span.End()

	span.SetAttributes(attribute.String("id", request.ID.String()))

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Project)

	err = tx.NewRaw(projectDeleteQuery, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrProjectDeleteNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
