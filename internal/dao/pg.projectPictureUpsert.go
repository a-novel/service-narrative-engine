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

//go:embed pg.projectPictureUpsert.sql
var projectPictureUpsertQuery string

// ProjectPictureUpsertRequest is the input to [PgProjectPictureUpsert.Exec].
type ProjectPictureUpsertRequest struct {
	ProjectID uuid.UUID
	// Picture names the picture: logo or background.
	Picture string

	// Data holds the encoded image.
	Data []byte
	// MediaType is the media type the picture is encoded in.
	MediaType string

	// Now is the instant stored as the picture's version.
	Now time.Time
}

// PgProjectPictureUpsert replaces a picture of a project, and its version, in one statement. It returns the
// project. A missing project fails the write.
type PgProjectPictureUpsert struct{}

func NewPgProjectPictureUpsert() *PgProjectPictureUpsert {
	return &PgProjectPictureUpsert{}
}

func (dao *PgProjectPictureUpsert) Exec(ctx context.Context, request *ProjectPictureUpsertRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectPictureUpsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Project{}

	err = db.NewRaw(
		projectPictureUpsertQuery, request.ProjectID, request.Picture, request.Data, request.MediaType, request.Now,
	).Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
