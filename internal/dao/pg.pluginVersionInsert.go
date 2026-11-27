package dao

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.pluginVersionInsert.sql
var pluginVersionInsertQuery string

// ErrPluginVersionExists is returned by [PgPluginVersionInsert.Exec] when the registry already holds the
// version.
var ErrPluginVersionExists = errors.New("plugin version exists")

// PluginVersionInsertLanguage holds the translations of a [PluginVersionInsertRequest] in one language.
type PluginVersionInsertLanguage struct {
	// Language is the language of the translations, as a BCP 47 tag.
	Language string

	// Title names the plugin in the list of plugins.
	Title string
	// Description says what the plugin does, in the list of plugins.
	Description string
	// Messages holds the translations of the plugin's interface and of its schema, by key.
	Messages json.RawMessage
}

// PluginVersionInsertRequest is the input to [PgPluginVersionInsert.Exec].
type PluginVersionInsertRequest struct {
	PluginVersionKey

	// Schema is the JSON Schema the content of an instance follows.
	Schema json.RawMessage
	// Exports holds the lists of linkable items an instance exposes, by name.
	Exports json.RawMessage
	// Migration is the JavaScript module that brings content from the previous version to this one. Nil when
	// the version needs none.
	Migration []byte
	// UI is the JavaScript module that renders the plugin.
	UI []byte

	// Proof is the token the publisher's API key signed for the version, a compact JWT. Nil for a version
	// published without a key.
	Proof *string
	// APIKeyID is the ID of the API key that signed Proof. Nil when Proof is.
	APIKeyID *string

	// Languages holds the translations of the version, each language once.
	Languages []PluginVersionInsertLanguage

	// Now is the instant stored as published_at.
	Now time.Time
}

// PgPluginVersionInsert stores one version of a plugin with its translations, in one statement: the registry
// holds the whole version or none of it. A version it already holds fails the write with
// [ErrPluginVersionExists] and stays as it is.
type PgPluginVersionInsert struct{}

func NewPgPluginVersionInsert() *PgPluginVersionInsert {
	return &PgPluginVersionInsert{}
}

func (dao *PgPluginVersionInsert) Exec(ctx context.Context, request *PluginVersionInsertRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgPluginVersionInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	languages := lo.Map(request.Languages, func(item PluginVersionInsertLanguage, _ int) string { return item.Language })
	titles := lo.Map(request.Languages, func(item PluginVersionInsertLanguage, _ int) string { return item.Title })
	descriptions := lo.Map(
		request.Languages, func(item PluginVersionInsertLanguage, _ int) string { return item.Description },
	)
	messages := lo.Map(
		request.Languages, func(item PluginVersionInsertLanguage, _ int) string { return string(item.Messages) },
	)

	_, err = db.NewRaw(
		pluginVersionInsertQuery,
		request.Plugin, request.Major, request.Minor, request.Patch, request.Prerelease,
		string(request.Schema), string(request.Exports), request.Migration, request.UI,
		request.Proof, request.APIKeyID, request.Now,
		bun.List(languages), bun.List(titles), bun.List(descriptions), bun.List(messages),
	).Exec(ctx)

	code, constraint := postgres.SQLState(err)
	if code == pgUniqueViolation && constraint == "plugin_versions_pkey" {
		err = errors.Join(err, ErrPluginVersionExists)
	}

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
