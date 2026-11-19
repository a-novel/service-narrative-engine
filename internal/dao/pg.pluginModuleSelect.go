package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.pluginModuleSelect.sql
var pluginModuleSelectQuery string

// PluginModule is one JavaScript module of a version of a plugin.
type PluginModule struct {
	// Code holds the module's source.
	Code []byte `bun:"code"`
	// Checksum is the SHA-256 of Code in lowercase hex.
	Checksum string `bun:"checksum"`

	// PublishedAt is when the module's version was published. A module never changes after it.
	PublishedAt time.Time `bun:"published_at"`
}

// PluginModuleSelectRequest is the input to [PgPluginModuleSelect.Exec].
type PluginModuleSelectRequest struct {
	PluginVersionKey

	// Module names the module to read: ui or migration.
	Module string
}

// PgPluginModuleSelect reads one module of a version of a plugin, with its checksum.
type PgPluginModuleSelect struct{}

func NewPgPluginModuleSelect() *PgPluginModuleSelect {
	return &PgPluginModuleSelect{}
}

func (dao *PgPluginModuleSelect) Exec(ctx context.Context, request *PluginModuleSelectRequest) (*PluginModule, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgPluginModuleSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &PluginModule{}

	err = db.NewRaw(
		pluginModuleSelectQuery,
		request.Plugin, request.Major, request.Minor, request.Patch, request.Prerelease, request.Module,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrPluginModuleNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
