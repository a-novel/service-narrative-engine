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

//go:embed pg.instanceUpdate.sql
var instanceUpdateQuery string

// InstanceUpdateRequest is the input to [PgInstanceUpdate.Exec].
type InstanceUpdateRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID

	// Name is the display name the writer gives the instance.
	Name string
}

// PgInstanceUpdate renames an instance of a project. It leaves the instance's updated_at as it was.
type PgInstanceUpdate struct{}

func NewPgInstanceUpdate() *PgInstanceUpdate {
	return &PgInstanceUpdate{}
}

func (dao *PgInstanceUpdate) Exec(ctx context.Context, request *InstanceUpdateRequest) (*InstanceDescriptor, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgInstanceUpdate")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &InstanceDescriptor{}

	err = db.NewRaw(instanceUpdateQuery, request.ProjectID, request.ID, request.Name).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrInstanceNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
