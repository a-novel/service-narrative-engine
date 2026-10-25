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

//go:embed pg.projectDelete.sql
var projectDeleteQuery string

// ErrProjectDeleteNotFound is returned by [PgProjectDelete.Exec] when no project with the ID belongs
// to the owner at the revision.
var ErrProjectDeleteNotFound = errors.New("project not found at this revision")

// ProjectDeleteRequest is the input to [PgProjectDelete.Exec].
type ProjectDeleteRequest struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	// Revision is the revision the caller read.
	Revision int
}

// PgProjectDelete removes a project with its scenes, layers and cells when the stored revision is the
// one the caller read.
type PgProjectDelete struct{}

func NewPgProjectDelete() *PgProjectDelete {
	return &PgProjectDelete{}
}

func (dao *PgProjectDelete) Exec(ctx context.Context, request *ProjectDeleteRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Project{}

	err = db.NewRaw(projectDeleteQuery, request.ID, request.OwnerID, request.Revision).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectDeleteNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
