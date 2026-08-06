package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
)

//go:embed pg.projectLock.sql
var projectLockQuery string

func lockProject(
	ctx context.Context,
	db bun.IDB,
	projectID uuid.UUID,
	ownerID uuid.UUID,
) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.lockProject")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", projectID.String()),
		attribute.String("project.owner_id", ownerID.String()),
	)

	var id uuid.UUID

	err := db.NewRaw(projectLockQuery, projectID, ownerID).Scan(ctx, &id)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrProjectLockNotFound)
	}

	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
