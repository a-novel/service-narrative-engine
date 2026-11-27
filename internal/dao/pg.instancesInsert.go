package dao

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.instancesInsert.sql
var instancesInsertQuery string

// ErrInstancesInsertFull is returned by [PgInstancesInsert.Exec] when the project would hold more instances
// than the request allows. The batch inserts nothing.
var ErrInstancesInsertFull = errors.New("project holds the most instances it can")

// InstancesInsertItem is one instance of an [InstancesInsertRequest].
type InstancesInsertItem struct {
	ID uuid.UUID

	// Plugin is the name of the plugin, as the registry holds it.
	Plugin string
	// Version is the exact version of the plugin the content follows, as a semantic version.
	Version string
	// Name is the display name the writer gave the instance.
	Name string

	// Content is the document the plugin edits.
	Content json.RawMessage
}

// InstancesInsertRequest is the input to [PgInstancesInsert.Exec].
type InstancesInsertRequest struct {
	// ProjectID is the project the instances join.
	ProjectID uuid.UUID

	// Instances lists the instances to create.
	Instances []InstancesInsertItem
	// Max is how many instances the project may hold once they joined it.
	Max int

	// Now is the instant stored as created_at and updated_at.
	Now time.Time
}

// PgInstancesInsert creates instances in a project, all of them or none, and returns them in the order of the
// request. The first one becomes the default instance of a project that has none.
type PgInstancesInsert struct{}

func NewPgInstancesInsert() *PgInstancesInsert {
	return &PgInstancesInsert{}
}

func (dao *PgInstancesInsert) Exec(
	ctx context.Context, request *InstancesInsertRequest,
) ([]*InstanceDescriptor, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgInstancesInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	ids := lo.Map(request.Instances, func(item InstancesInsertItem, _ int) uuid.UUID { return item.ID })
	plugins := lo.Map(request.Instances, func(item InstancesInsertItem, _ int) string { return item.Plugin })
	versions := lo.Map(request.Instances, func(item InstancesInsertItem, _ int) string { return item.Version })
	names := lo.Map(request.Instances, func(item InstancesInsertItem, _ int) string { return item.Name })
	contents := lo.Map(request.Instances, func(item InstancesInsertItem, _ int) string { return string(item.Content) })

	var entities []*InstanceDescriptor

	err = db.NewRaw(
		instancesInsertQuery,
		request.ProjectID, bun.List(ids), bun.List(plugins), bun.List(versions), bun.List(names),
		bun.List(contents), request.Now, request.Max,
	).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	// The statement inserts every instance or none, and only the cap makes it insert none.
	if len(entities) < len(request.Instances) {
		return nil, otel.ReportError(span, ErrInstancesInsertFull)
	}

	return entities, nil
}
