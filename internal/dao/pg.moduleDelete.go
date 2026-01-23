package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.moduleDelete.sql
var moduleDeleteQuery string

var ErrModuleDeleteNotFound = errors.New("module not found")

type ModuleDeleteRequest struct {
	ID         string
	Namespace  string
	Version    string
	Preversion string
}

type ModuleDelete struct{}

func NewModuleDelete() *ModuleDelete {
	return new(ModuleDelete)
}

func (repository *ModuleDelete) Exec(ctx context.Context, request *ModuleDeleteRequest) (*Module, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ModuleDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("id", request.ID),
		attribute.String("namespace", request.Namespace),
		attribute.String("version", request.Version),
		attribute.String("preversion", request.Preversion),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Module)

	err = tx.
		NewRaw(moduleDeleteQuery, request.ID, request.Namespace, request.Version, request.Preversion).
		Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrModuleDeleteNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
