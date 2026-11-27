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

//go:embed pg.instanceContentUpdate.sql
var instanceContentUpdateQuery string

// ErrInstanceContentUpdateStale is returned by [PgInstanceContentUpdate.Exec] when the instance no longer holds
// the updated_at the request names. The write changes nothing.
var ErrInstanceContentUpdateStale = errors.New("instance changed since it was read")

// InstanceContentUpdateRequest is the input to [PgInstanceContentUpdate.Exec].
type InstanceContentUpdateRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID

	// Version is the exact version of the plugin the content follows, as a semantic version.
	Version string
	// Content is the document the plugin edits.
	Content json.RawMessage

	// UpdatedAt is the updated_at the instance held when the caller read it.
	UpdatedAt time.Time
	// Now is the instant stored as updated_at.
	Now time.Time
}

// instanceContentUpdateRow is the instance a write targets, as the write left it, and whether the write
// applied.
type instanceContentUpdateRow struct {
	InstanceDescriptor

	// Stale is true when the instance held another updated_at than the request names.
	Stale bool `bun:"stale"`
}

// PgInstanceContentUpdate replaces the version and the content of an instance in one statement, only while the
// instance still holds the updated_at the request names. It returns the instance without its content.
type PgInstanceContentUpdate struct{}

func NewPgInstanceContentUpdate() *PgInstanceContentUpdate {
	return &PgInstanceContentUpdate{}
}

func (dao *PgInstanceContentUpdate) Exec(
	ctx context.Context, request *InstanceContentUpdateRequest,
) (*InstanceDescriptor, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgInstanceContentUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var row instanceContentUpdateRow

	err = db.NewRaw(
		instanceContentUpdateQuery,
		request.ProjectID, request.ID, request.Version, string(request.Content), request.UpdatedAt, request.Now,
	).Scan(ctx, &row)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrInstanceNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	if row.Stale {
		return nil, otel.ReportError(span, ErrInstanceContentUpdateStale)
	}

	return &row.InstanceDescriptor, nil
}
