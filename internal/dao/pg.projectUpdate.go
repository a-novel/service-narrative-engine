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

// ErrProjectUpdateNotFound is returned by [PgProjectUpdate.Exec] when no project has the ID.
var ErrProjectUpdateNotFound = errors.New("project not found")

// ProjectUpdateRequest is the input to [PgProjectUpdate.Exec].
type ProjectUpdateRequest struct {
	ID uuid.UUID

	// Title names the narrative.
	Title string
	// Premise says what the narrative is about, in a few sentences.
	Premise string
	// Language is the language the project is written in: en or fr.
	Language string

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgProjectUpdate replaces a project's own fields.
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
		request.ID, request.Title, request.Premise, request.Language, request.Now,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectUpdateNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
