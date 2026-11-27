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

//go:embed pg.projectLinksSelect.sql
var projectLinksSelectQuery string

// ErrProjectLinksSelectNotFound is returned by [PgProjectLinksSelect.Exec] when no project has the ID.
var ErrProjectLinksSelectNotFound = errors.New("project not found")

// ProjectLinksSelectRequest is the input to [PgProjectLinksSelect.Exec].
type ProjectLinksSelectRequest struct {
	ProjectID uuid.UUID
}

// PgProjectLinksSelect reads the links of a project. A project that never wrote its links returns an empty
// list with a nil UpdatedAt.
type PgProjectLinksSelect struct{}

func NewPgProjectLinksSelect() *PgProjectLinksSelect {
	return &PgProjectLinksSelect{}
}

func (dao *PgProjectLinksSelect) Exec(ctx context.Context, request *ProjectLinksSelectRequest) (*ProjectLinks, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectLinksSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &ProjectLinks{}

	err = db.NewRaw(projectLinksSelectQuery, request.ProjectID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectLinksSelectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
