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

//go:embed pg.instanceDelete.sql
var instanceDeleteQuery string

// ErrInstanceDeleteDefault is returned by [PgInstanceDelete.Exec] when the instance is the default instance of
// its project. The delete removes nothing.
var ErrInstanceDeleteDefault = errors.New("instance is the default of its project")

// InstanceDeleteRequest is the input to [PgInstanceDelete.Exec].
type InstanceDeleteRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID
}

// PgInstanceDelete removes an instance of a project and its content, and returns it without its content. It
// refuses the default instance of the project.
type PgInstanceDelete struct{}

func NewPgInstanceDelete() *PgInstanceDelete {
	return &PgInstanceDelete{}
}

func (dao *PgInstanceDelete) Exec(ctx context.Context, request *InstanceDeleteRequest) (*InstanceDescriptor, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgInstanceDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &InstanceDescriptor{}

	err = db.NewRaw(instanceDeleteQuery, request.ProjectID, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrInstanceNotFound)
	}

	code, constraint := postgres.SQLState(err)
	if code == pgRestrictViolation && constraint == "projects_default_instance_fkey" {
		err = errors.Join(err, ErrInstanceDeleteDefault)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
