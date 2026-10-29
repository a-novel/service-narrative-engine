package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.beatsDelete.sql
var beatsDeleteQuery string

// BeatsDeleteRequest is the input to [PgBeatsDelete.Exec].
type BeatsDeleteRequest struct {
	ProjectID uuid.UUID
	// IDs lists the beats to delete.
	IDs []uuid.UUID
}

// PgBeatsDelete removes beats of a project with their scenes and cells, and moves the beats after them up
// to close the gaps. It returns the deleted beats, in no particular order, and skips an ID the project does not
// hold.
type PgBeatsDelete struct{}

func NewPgBeatsDelete() *PgBeatsDelete {
	return &PgBeatsDelete{}
}

func (dao *PgBeatsDelete) Exec(ctx context.Context, request *BeatsDeleteRequest) ([]*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgBeatsDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*Beat

	err = db.NewRaw(beatsDeleteQuery, request.ProjectID, bun.List(request.IDs)).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("beats.count", len(entities)))

	return entities, nil
}
