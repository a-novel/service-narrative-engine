package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectLinksUpsert.sql
var projectLinksUpsertQuery string

// ErrProjectLinksUpsertStale is returned by [PgProjectLinksUpsert.Exec] when the links no longer hold the
// updated_at the request names. The write changes nothing.
var ErrProjectLinksUpsertStale = errors.New("links changed since they were read")

// ProjectLinksUpsertRequest is the input to [PgProjectLinksUpsert.Exec].
type ProjectLinksUpsertRequest struct {
	ProjectID uuid.UUID

	// Links is a JSON array of links, each a source, a destination and their relationship.
	Links json.RawMessage

	// UpdatedAt is the updated_at the links held when the caller read them, nil when the project had never
	// written its links.
	UpdatedAt *time.Time
	// Now is the instant stored as updated_at.
	Now time.Time
}

// ProjectLinksUpsertRow is when [PgProjectLinksUpsert.Exec] wrote the links. The caller already holds them.
type ProjectLinksUpsertRow struct {
	UpdatedAt time.Time `bun:"updated_at"`
}

// PgProjectLinksUpsert writes the links of a project in one statement. It replaces links already written only
// while they still hold the updated_at the request names, and joins [ErrProjectLinksUpsertStale] otherwise. A
// project that never wrote its links takes them whatever the request names. A missing project fails the write.
type PgProjectLinksUpsert struct{}

func NewPgProjectLinksUpsert() *PgProjectLinksUpsert {
	return &PgProjectLinksUpsert{}
}

func (dao *PgProjectLinksUpsert) Exec(
	ctx context.Context, request *ProjectLinksUpsertRequest,
) (*ProjectLinksUpsertRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectLinksUpsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &ProjectLinksUpsertRow{}

	err = db.NewRaw(
		projectLinksUpsertQuery, request.ProjectID, string(request.Links), request.UpdatedAt, request.Now,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectLinksUpsertStale)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
