package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.scenesInsert.sql
var scenesInsertQuery string

var (
	// ErrScenesInsertFull is returned by [PgScenesInsert.Exec] when a scene would take a position past 9,
	// so the beat would hold more than the 10 scenes the schema allows.
	ErrScenesInsertFull = errors.New("beat holds the most scenes it can")
	// ErrScenesInsertBeatNotFound is returned by [PgScenesInsert.Exec] when the project holds no beat with
	// the ID.
	ErrScenesInsertBeatNotFound = errors.New("beat not found")
)

// ScenesInsertItem is one scene of a [ScenesInsertRequest].
type ScenesInsertItem struct {
	ID uuid.UUID

	// Title names the scene in the composer.
	Title string
	// Description refines the beat's description for the scene.
	Description string
	// Prompt refines the beat's prompt for the scene.
	Prompt string
}

// ScenesInsertRequest is the input to [PgScenesInsert.Exec].
type ScenesInsertRequest struct {
	// ProjectID is the project the scenes join.
	ProjectID uuid.UUID
	// BeatID is the beat the scenes join.
	BeatID uuid.UUID

	// Position is the position of the first scene in the beat, and the others follow it in order. A
	// position past the beat's last scene appends.
	Position int
	// Scenes lists the scenes to create, in order.
	Scenes []ScenesInsertItem

	// Now is the instant stored as updated_at.
	Now time.Time
}

// PgScenesInsert creates scenes in one beat from Position on, and moves the beat's scenes at or after
// Position down to make room. It returns the new scenes in no particular order.
type PgScenesInsert struct{}

func NewPgScenesInsert() *PgScenesInsert {
	return &PgScenesInsert{}
}

func (dao *PgScenesInsert) Exec(ctx context.Context, request *ScenesInsertRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgScenesInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	ids := lo.Map(request.Scenes, func(item ScenesInsertItem, _ int) uuid.UUID { return item.ID })
	titles := lo.Map(request.Scenes, func(item ScenesInsertItem, _ int) string { return item.Title })
	descriptions := lo.Map(request.Scenes, func(item ScenesInsertItem, _ int) string { return item.Description })
	prompts := lo.Map(request.Scenes, func(item ScenesInsertItem, _ int) string { return item.Prompt })

	var entities []*Scene

	err = db.NewRaw(
		scenesInsertQuery,
		request.ProjectID, request.BeatID, request.Position, bun.List(ids), bun.List(titles),
		bun.List(descriptions), bun.List(prompts), request.Now,
	).Scan(ctx, &entities)

	switch {
	case isViolation(err, pgCheckViolation, "scenes_position_check"):
		err = errors.Join(err, ErrScenesInsertFull)
	case isViolation(err, pgForeignKeyViolation, "scenes_beat_fkey"):
		err = errors.Join(err, ErrScenesInsertBeatNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("scenes.count", len(entities)))

	return entities, nil
}
