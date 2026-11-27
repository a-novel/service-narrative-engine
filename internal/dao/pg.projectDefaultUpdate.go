package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectDefaultUpdate.sql
var projectDefaultUpdateQuery string

// ProjectDefaultUpdateRequest is the input to [PgProjectDefaultUpdate.Exec].
type ProjectDefaultUpdateRequest struct {
	ProjectID uuid.UUID
	// InstanceID is the instance that becomes the default of the project.
	InstanceID uuid.UUID

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgProjectDefaultUpdate makes an instance of a project its default, and returns the project. It joins
// [ErrProjectNotFound] when no project has the ID, and [ErrInstanceNotFound] when the project holds no such
// instance.
type PgProjectDefaultUpdate struct{}

func NewPgProjectDefaultUpdate() *PgProjectDefaultUpdate {
	return &PgProjectDefaultUpdate{}
}

func (dao *PgProjectDefaultUpdate) Exec(ctx context.Context, request *ProjectDefaultUpdateRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectDefaultUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Project{}

	err = db.NewRaw(projectDefaultUpdateQuery, request.ProjectID, request.InstanceID, request.Now).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectNotFound)
	}

	code, constraint := postgres.SQLState(err)
	if code == pgForeignKeyViolation && constraint == "projects_default_instance_fkey" {
		err = errors.Join(err, ErrInstanceNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
