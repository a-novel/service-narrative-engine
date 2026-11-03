package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectBytes.sql
var projectBytesQuery string

// ErrProjectBytesNotFound is returned when no project matches the requested ID.
var ErrProjectBytesNotFound = errors.New("project not found")

// ProjectBytesRequest is the input to [PgProjectBytes.Exec].
type ProjectBytesRequest struct {
	ID uuid.UUID
}

// ProjectBytes is how many bytes a project stores.
type ProjectBytes struct {
	// StoredBytes adds the bytes of the project's own row to those of every row it holds.
	StoredBytes int64 `bun:"stored_bytes"`
}

// PgProjectBytes reads how many bytes a project stores. Every write to a row the project holds keeps the
// project's count in the statement that changes the bytes, so a read in the write's transaction sees the
// write.
type PgProjectBytes struct{}

func NewPgProjectBytes() *PgProjectBytes {
	return &PgProjectBytes{}
}

func (dao *PgProjectBytes) Exec(ctx context.Context, request *ProjectBytesRequest) (*ProjectBytes, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectBytes")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entity ProjectBytes

	err = db.NewRaw(projectBytesQuery, request.ID).Scan(ctx, &entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectBytesNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int64("project.stored_bytes", entity.StoredBytes))

	return &entity, nil
}
