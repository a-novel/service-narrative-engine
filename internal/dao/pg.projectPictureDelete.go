package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectPictureDelete.sql
var projectPictureDeleteQuery string

// ProjectPictureDeleteRequest is the input to [PgProjectPictureDelete.Exec].
type ProjectPictureDeleteRequest struct {
	ProjectID uuid.UUID
	// Picture names the picture: logo or background.
	Picture string
}

// PgProjectPictureDelete removes a picture of a project, and its version, in one statement. It returns the
// project. A project without the picture keeps none, and a missing project fails the write.
type PgProjectPictureDelete struct{}

func NewPgProjectPictureDelete() *PgProjectPictureDelete {
	return &PgProjectPictureDelete{}
}

func (dao *PgProjectPictureDelete) Exec(ctx context.Context, request *ProjectPictureDeleteRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectPictureDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Project{}

	err = db.NewRaw(projectPictureDeleteQuery, request.ProjectID, request.Picture).Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
