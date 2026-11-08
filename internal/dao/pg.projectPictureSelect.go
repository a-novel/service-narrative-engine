package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectPictureSelect.sql
var projectPictureSelectQuery string

// ErrProjectPictureSelectNotFound is returned by [PgProjectPictureSelect.Exec] when the project holds no
// such picture.
var ErrProjectPictureSelectNotFound = errors.New("picture not found")

// ProjectPictureSelectRequest is the input to [PgProjectPictureSelect.Exec].
type ProjectPictureSelectRequest struct {
	ProjectID uuid.UUID
	// Picture names the picture: logo or background.
	Picture string
}

// PgProjectPictureSelect reads a picture of a project, with its version. The caller checks the owner with
// [PgProjectOwned].
type PgProjectPictureSelect struct{}

func NewPgProjectPictureSelect() *PgProjectPictureSelect {
	return &PgProjectPictureSelect{}
}

func (dao *PgProjectPictureSelect) Exec(
	ctx context.Context, request *ProjectPictureSelectRequest,
) (*ProjectPicture, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectPictureSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &ProjectPicture{}

	err = db.NewRaw(projectPictureSelectQuery, request.ProjectID, request.Picture).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectPictureSelectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
