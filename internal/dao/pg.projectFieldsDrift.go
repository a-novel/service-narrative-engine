package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectFieldsDrift.sql
var projectFieldsDriftQuery string

// ProjectFieldsDriftRequest is the input to [PgProjectFieldsDrift.Exec].
type ProjectFieldsDriftRequest struct {
	// OwnerID is the user whose projects to compare.
	OwnerID uuid.UUID

	// Projects maps the ID of each project a client holds to the UpdatedAt it holds, as RFC 3339 text.
	Projects map[string]string

	// Limit caps the rows.
	Limit int
}

// ProjectFieldsDriftRow is one drift: a project of the owner that drifted, or a held ID that names no project of
// the owner.
type ProjectFieldsDriftRow struct {
	// Project is the drifted project, with no logo. It is empty on a removed row.
	Project

	// RemovedID is the held ID that names no project of the owner, nil on the row of a drifted project.
	RemovedID *uuid.UUID `bun:"removed_id,type:uuid"`
}

// PgProjectFieldsDrift compares the projects a client holds with an owner's, by their UpdatedAt, in one query. It
// returns every project of the owner the client holds at another instant or not at all, and every held ID that
// names no project of the owner.
type PgProjectFieldsDrift struct{}

func NewPgProjectFieldsDrift() *PgProjectFieldsDrift {
	return &PgProjectFieldsDrift{}
}

func (dao *PgProjectFieldsDrift) Exec(
	ctx context.Context, request *ProjectFieldsDriftRequest,
) ([]*ProjectFieldsDriftRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectFieldsDrift")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*ProjectFieldsDriftRow

	err = db.NewRaw(projectFieldsDriftQuery, request.OwnerID, request.Projects, request.Limit).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("projects.drifted", len(entities)))

	return entities, nil
}
