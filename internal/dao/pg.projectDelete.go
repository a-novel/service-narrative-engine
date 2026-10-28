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
// to the owner.
var ErrProjectDeleteNotFound = errors.New("project not found")

// ProjectDeleteRequest is the input to [PgProjectDelete.Exec].
type ProjectDeleteRequest struct {
	ID uuid.UUID
	// OwnerID is the user who must own the project. Another owner's project reads as not found.
	OwnerID uuid.UUID
}

// PgProjectDelete removes a project with its scenes, layers and cells.
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

	err = db.NewRaw(projectDeleteQuery, request.ID, request.OwnerID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectDeleteNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
