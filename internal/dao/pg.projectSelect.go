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

//go:embed pg.projectSelect.sql
var projectSelectQuery string

var ErrProjectSelectNotFound = errors.New("project not found")

type ProjectSelectRequest struct {
	ID uuid.UUID
}

type ProjectSelect struct{}

func NewProjectSelect() *ProjectSelect {
	return new(ProjectSelect)
}

func (repository *ProjectSelect) Exec(ctx context.Context, request *ProjectSelectRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ProjectSelect")
	defer span.End()

	span.SetAttributes(attribute.String("id", request.ID.String()))

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Project)

	err = tx.NewRaw(projectSelectQuery, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrProjectSelectNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
