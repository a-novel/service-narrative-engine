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

//go:embed pg.instanceLinksUpdate.sql
var instanceLinksUpdateQuery string

// ErrInstanceLinksUpdateStale is returned by [PgInstanceLinksUpdate.Exec] when the instance no longer holds the
// links_updated_at the request names. The write changes nothing.
var ErrInstanceLinksUpdateStale = errors.New("links changed since they were read")

// InstanceLinksUpdateRequest is the input to [PgInstanceLinksUpdate.Exec].
type InstanceLinksUpdateRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID

	// Links is a JSON array of links, each a source, a destination and their relationship.
	Links json.RawMessage

	// UpdatedAt is the links_updated_at the instance held when the caller read its links, nil when the instance
	// had never written them.
	UpdatedAt *time.Time
	// Now is the instant stored as links_updated_at.
	Now time.Time
}

// InstanceLinksUpdateRow is when [PgInstanceLinksUpdate.Exec] wrote the links. The caller already holds them.
type InstanceLinksUpdateRow struct {
	UpdatedAt time.Time
}

// instanceLinksUpdateRow is the stamp a write left on the links of the instance it targets, and whether the
// write applied.
type instanceLinksUpdateRow struct {
	// UpdatedAt is the stamp of the write, nil when it did not apply.
	UpdatedAt *time.Time `bun:"links_updated_at"`
	// Stale is true when the instance held another links_updated_at than the request names.
	Stale bool `bun:"stale"`
}

// PgInstanceLinksUpdate replaces the links of an instance in one statement, only while the instance still holds
// the links_updated_at the request names. It leaves the content and its stamp as they were.
type PgInstanceLinksUpdate struct{}

func NewPgInstanceLinksUpdate() *PgInstanceLinksUpdate {
	return &PgInstanceLinksUpdate{}
}

func (dao *PgInstanceLinksUpdate) Exec(
	ctx context.Context, request *InstanceLinksUpdateRequest,
) (*InstanceLinksUpdateRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgInstanceLinksUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var row instanceLinksUpdateRow

	err = db.NewRaw(
		instanceLinksUpdateQuery,
		request.ProjectID, request.ID, string(request.Links), request.UpdatedAt, request.Now,
	).Scan(ctx, &row)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrInstanceNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	if row.Stale {
		return nil, otel.ReportError(span, ErrInstanceLinksUpdateStale)
	}

	return &InstanceLinksUpdateRow{UpdatedAt: *row.UpdatedAt}, nil
}
