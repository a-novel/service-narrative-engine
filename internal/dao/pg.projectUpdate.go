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

//go:embed pg.projectUpdate.sql
var projectUpdateQuery string

// ErrProjectUpdateNotFound is returned by [PgProjectUpdate.Exec] when no project with the ID belongs
// to the owner at the revision.
var ErrProjectUpdateNotFound = errors.New("project not found at this revision")

// ProjectUpdateRequest is the input to [PgProjectUpdate.Exec].
type ProjectUpdateRequest struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	// Revision is the revision the caller read.
	Revision int
	Title    string
	Premise  string
	Language string
	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgProjectUpdate replaces a project's own fields when the stored revision is the one the caller read.
type PgProjectUpdate struct{}

func NewPgProjectUpdate() *PgProjectUpdate {
	return &PgProjectUpdate{}
}

func (dao *PgProjectUpdate) Exec(ctx context.Context, request *ProjectUpdateRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Project{}

	err = db.NewRaw(
		projectUpdateQuery,
		request.ID, request.OwnerID, request.Revision, request.Title,
		request.Premise, request.Language, request.Now,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectUpdateNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
