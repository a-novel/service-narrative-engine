package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectEditsList.sql
var projectEditsListQuery string

// ProjectEditsListRequest is the input to [PgProjectEditsList.Exec].
type ProjectEditsListRequest struct {
	// OwnerID is the user whose projects to read.
	OwnerID uuid.UUID

	// Limit caps the projects.
	Limit int
}

// ProjectEditsListRow is when one project of the owner last changed.
type ProjectEditsListRow struct {
	ID uuid.UUID `bun:"id,type:uuid"`

	EditedAt time.Time `bun:"edited_at"`
}

// PgProjectEditsList reads the EditedAt of every project an owner holds, in no order.
type PgProjectEditsList struct{}

func NewPgProjectEditsList() *PgProjectEditsList {
	return &PgProjectEditsList{}
}

func (dao *PgProjectEditsList) Exec(
	ctx context.Context, request *ProjectEditsListRequest,
) ([]*ProjectEditsListRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectEditsList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*ProjectEditsListRow

	err = db.NewRaw(projectEditsListQuery, request.OwnerID, request.Limit).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("projects.count", len(entities)))

	return entities, nil
}
