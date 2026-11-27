package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.instanceSelect.sql
var instanceSelectQuery string

// InstanceSelectRequest is the input to [PgInstanceSelect.Exec].
type InstanceSelectRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID
}

// PgInstanceSelect reads an instance of a project, with its content.
type PgInstanceSelect struct{}

func NewPgInstanceSelect() *PgInstanceSelect {
	return &PgInstanceSelect{}
}

func (dao *PgInstanceSelect) Exec(ctx context.Context, request *InstanceSelectRequest) (*Instance, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgInstanceSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Instance{}

	err = db.NewRaw(instanceSelectQuery, request.ProjectID, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrInstanceNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
