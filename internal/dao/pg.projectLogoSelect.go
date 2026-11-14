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

//go:embed pg.projectLogoSelect.sql
var projectLogoSelectQuery string

// ErrProjectLogoSelectNotFound is returned by [PgProjectLogoSelect.Exec] when the project holds no logo.
var ErrProjectLogoSelectNotFound = errors.New("project logo not found")

// ProjectLogoSelectRequest is the input to [PgProjectLogoSelect.Exec].
type ProjectLogoSelectRequest struct {
	ProjectID uuid.UUID
}

// PgProjectLogoSelect reads the logo of a project, with its checksum. The caller checks the owner with
// [PgProjectOwned].
type PgProjectLogoSelect struct{}

func NewPgProjectLogoSelect() *PgProjectLogoSelect {
	return &PgProjectLogoSelect{}
}

func (dao *PgProjectLogoSelect) Exec(
	ctx context.Context, request *ProjectLogoSelectRequest,
) (*ProjectLogo, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectLogoSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &ProjectLogo{}

	err = db.NewRaw(projectLogoSelectQuery, request.ProjectID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectLogoSelectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
