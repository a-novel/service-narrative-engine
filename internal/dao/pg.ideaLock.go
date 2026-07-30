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

//go:embed pg.ideaLock.sql
var ideaLockQuery string

func lockIdea(
	ctx context.Context,
	db bun.IDB,
	ideaID uuid.UUID,
	ownerID uuid.UUID,
) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.lockIdea")
	defer span.End()

	span.SetAttributes(
		attribute.String("idea.id", ideaID.String()),
		attribute.String("idea.owner_id", ownerID.String()),
	)

	var id uuid.UUID

	err := db.NewRaw(ideaLockQuery, ideaID, ownerID).Scan(ctx, &id)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrIdeaLockNotFound)
	}

	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
