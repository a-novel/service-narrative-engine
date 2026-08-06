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

//go:embed pg.projectSelect.sql
var projectSelectQuery string

// ErrProjectSelectNotFound is returned for both absent and cross-owner Projects.
var ErrProjectSelectNotFound = errors.New("project not found")

// ProjectSelectRequest identifies an owner-scoped Project.
type ProjectSelectRequest struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
}

// PgProjectSelect reads only stable Project ownership metadata.
type PgProjectSelect struct{}

// NewPgProjectSelect creates an owner-scoped Project select operation.
func NewPgProjectSelect() *PgProjectSelect {
	return &PgProjectSelect{}
}

// Exec returns the owned Project without reading any content version.
func (operation *PgProjectSelect) Exec(
	ctx context.Context,
	request *ProjectSelectRequest,
) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectSelect")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var project Project

	err = db.NewRaw(projectSelectQuery, request.ID, request.OwnerID).Scan(ctx, &project)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectSelectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &project), nil
}
