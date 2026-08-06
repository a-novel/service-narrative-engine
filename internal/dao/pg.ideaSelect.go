package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.ideaSelect.sql
var ideaSelectQuery string

// ErrIdeaSelectNotFound is returned when the owner cannot access the requested Project Idea.
var ErrIdeaSelectNotFound = errors.New("idea not found")

// IdeaSelectRequest identifies a Project whose current Idea Version is requested.
type IdeaSelectRequest struct {
	// ProjectID identifies the Project.
	ProjectID uuid.UUID
	// OwnerID identifies the user requesting its Idea.
	OwnerID uuid.UUID
}

// PgIdeaSelect retrieves the current Idea Version within its Project owner boundary.
type PgIdeaSelect struct{}

// NewPgIdeaSelect creates an owner-scoped current-Idea select operation.
func NewPgIdeaSelect() *PgIdeaSelect {
	return &PgIdeaSelect{}
}

// Exec returns the current owned Idea or [ErrIdeaSelectNotFound].
func (operation *PgIdeaSelect) Exec(ctx context.Context, request *IdeaSelectRequest) (*Idea, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgIdeaSelect")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var idea Idea

	err = db.NewRaw(ideaSelectQuery, request.ProjectID, request.OwnerID).Scan(ctx, &idea)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrIdeaSelectNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &idea), nil
}
