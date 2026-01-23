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

//go:embed pg.moduleSelect.sql
var moduleSelectQuery string

var ErrModuleSelectNotFound = errors.New("module not found")

type ModuleSelectRequest struct {
	ID         string
	Namespace  string
	Version    string
	Preversion string
}

type ModuleSelect struct{}

func NewModuleSelect() *ModuleSelect {
	return new(ModuleSelect)
}

func (repository *ModuleSelect) Exec(ctx context.Context, request *ModuleSelectRequest) (*Module, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ModuleSelect")
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
		NewRaw(moduleSelectQuery, request.ID, request.Namespace, request.Version, request.Preversion).
		Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrModuleSelectNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
