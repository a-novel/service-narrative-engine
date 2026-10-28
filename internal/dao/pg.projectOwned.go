package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectOwned.sql
var projectOwnedQuery string

// ProjectOwnedRequest is the input to [PgProjectOwned.Exec].
type ProjectOwnedRequest struct {
	ID uuid.UUID
	// OwnerID is the user who claims the project.
	OwnerID uuid.UUID
}

// PgProjectOwned reports whether a user owns a project. A missing project yields false with a nil
// error.
type PgProjectOwned struct{}

func NewPgProjectOwned() *PgProjectOwned {
	return &PgProjectOwned{}
}

func (dao *PgProjectOwned) Exec(ctx context.Context, request *ProjectOwnedRequest) (bool, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectOwned")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	res, err := db.NewRaw(projectOwnedQuery, request.ID, request.OwnerID).Exec(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	return n > 0, nil
}
