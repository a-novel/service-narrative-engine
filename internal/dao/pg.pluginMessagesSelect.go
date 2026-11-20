package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.pluginMessagesSelect.sql
var pluginMessagesSelectQuery string

// PluginMessagesSelectRequest is the input to [PgPluginMessagesSelect.Exec].
type PluginMessagesSelectRequest struct {
	PluginVersionKey

	// Language is the language to read, as a BCP 47 tag.
	Language string
}

// PgPluginMessagesSelect reads the translations of one version of a plugin in one language.
type PgPluginMessagesSelect struct{}

func NewPgPluginMessagesSelect() *PgPluginMessagesSelect {
	return &PgPluginMessagesSelect{}
}

func (dao *PgPluginMessagesSelect) Exec(
	ctx context.Context, request *PluginMessagesSelectRequest,
) (*PluginMessages, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgPluginMessagesSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &PluginMessages{}

	err = db.NewRaw(
		pluginMessagesSelectQuery,
		request.Plugin, request.Major, request.Minor, request.Patch, request.Prerelease, request.Language,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrPluginMessagesNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
