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

// ErrIdeaSelectNotFound is returned when the owner cannot access the requested Idea.
var ErrIdeaSelectNotFound = errors.New("idea not found")

// IdeaSelectRequest identifies an owner-scoped Idea for [PgIdeaSelect.Exec].
type IdeaSelectRequest struct {
	// ID identifies the Idea.
	ID uuid.UUID
	// OwnerID identifies the user requesting the Idea.
	OwnerID uuid.UUID
}

// PgIdeaSelect retrieves an Idea within its owner boundary.
type PgIdeaSelect struct{}

// NewPgIdeaSelect creates an owner-scoped Idea select operation.
func NewPgIdeaSelect() *PgIdeaSelect {
	return &PgIdeaSelect{}
}

// Exec returns the owned Idea or [ErrIdeaSelectNotFound].
func (operation *PgIdeaSelect) Exec(ctx context.Context, request *IdeaSelectRequest) (*Idea, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgIdeaSelect")
	defer span.End()

	span.SetAttributes(
		attribute.String("idea.id", request.ID.String()),
		attribute.String("idea.owner_id", request.OwnerID.String()),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var idea Idea

	err = db.NewRaw(ideaSelectQuery, request.ID, request.OwnerID).Scan(ctx, &idea)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrIdeaSelectNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &idea), nil
}
