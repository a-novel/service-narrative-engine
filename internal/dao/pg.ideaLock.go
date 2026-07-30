package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

//go:embed pg.ideaLock.sql
var ideaLockQuery string

func lockIdea(
	ctx context.Context,
	db bun.IDB,
	ideaID uuid.UUID,
	ownerID uuid.UUID,
) error {
	var id uuid.UUID

	err := db.NewRaw(ideaLockQuery, ideaID, ownerID).Scan(ctx, &id)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrIdeaLockNotFound)
	}

	return err
}
