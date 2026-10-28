package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectInsert.sql
var projectInsertQuery string

// ProjectInsertRequest is the input to [PgProjectInsert.Exec].
type ProjectInsertRequest struct {
	ID uuid.UUID
	// OwnerID is the user who owns the project, from the authentication service.
	OwnerID uuid.UUID

	// Title names the narrative.
	Title string
	// Premise says what the narrative is about, in a few sentences.
	Premise string
	// Language is the language the project is written in: en or fr.
	Language string

	// Now is the instant stored as both created_at and updated_at.
	Now time.Time
}

// PgProjectInsert creates a project.
type PgProjectInsert struct{}

func NewPgProjectInsert() *PgProjectInsert {
	return &PgProjectInsert{}
}

func (dao *PgProjectInsert) Exec(ctx context.Context, request *ProjectInsertRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Project{}

	err = db.NewRaw(
		projectInsertQuery, request.ID, request.OwnerID, request.Title, request.Premise, request.Language, request.Now,
	).Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
