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

//go:embed pg.projectSelect.sql
var projectSelectQuery string

// ErrProjectSelectNotFound is returned by [PgProjectSelect.Exec] when no project has the ID.
var ErrProjectSelectNotFound = errors.New("project not found")

// ProjectSelectRequest is the input to [PgProjectSelect.Exec].
type ProjectSelectRequest struct {
	ID uuid.UUID
}

// PgProjectSelect reads a project by its ID. The caller checks the owner.
type PgProjectSelect struct{}

func NewPgProjectSelect() *PgProjectSelect {
	return &PgProjectSelect{}
}

func (dao *PgProjectSelect) Exec(ctx context.Context, request *ProjectSelectRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Project{}

	err = db.NewRaw(projectSelectQuery, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectSelectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
