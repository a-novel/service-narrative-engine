package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/models"
)

//go:embed pg.moduleInsert.sql
var moduleInsertQuery string

var ErrModuleInsertAlreadyExists = errors.New("module already exists")

type ModuleInsertRequest struct {
	ID          string
	Namespace   string
	Version     string
	Preversion  string
	Description string
	Schema      jsonschema.Schema
	UI          models.ModuleUi
	Now         time.Time
}

type ModuleInsert struct{}

func NewModuleInsert() *ModuleInsert {
	return new(ModuleInsert)
}

func (repository *ModuleInsert) Exec(ctx context.Context, request *ModuleInsertRequest) (*Module, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ModuleInsert")
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

	err = tx.NewRaw(
		moduleInsertQuery,
		request.ID,
		request.Namespace,
		request.Version,
		request.Preversion,
		request.Description,
		request.Schema,
		request.UI,
		request.Now,
	).Scan(ctx, entity)
	if err != nil {
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
			err = errors.Join(err, ErrModuleInsertAlreadyExists)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
