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

//go:embed pg.projectLogoUpsert.sql
var projectLogoUpsertQuery string

// ProjectLogoUpsertRequest is the input to [PgProjectLogoUpsert.Exec].
type ProjectLogoUpsertRequest struct {
	ProjectID uuid.UUID

	// Data holds the encoded image.
	Data []byte
	// MediaType is the media type the logo is encoded in.
	MediaType string

	// Now is the instant stored as the logo's version.
	Now time.Time
}

// PgProjectLogoUpsert replaces the logo of a project, and its version, in one statement. It returns the
// project. A missing project fails the write.
type PgProjectLogoUpsert struct{}

func NewPgProjectLogoUpsert() *PgProjectLogoUpsert {
	return &PgProjectLogoUpsert{}
}

func (dao *PgProjectLogoUpsert) Exec(ctx context.Context, request *ProjectLogoUpsertRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectLogoUpsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Project{}

	err = db.NewRaw(
		projectLogoUpsertQuery, request.ProjectID, request.Data, request.MediaType, request.Now,
	).Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
