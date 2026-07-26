package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.engineVersionSelect.sql
var engineVersionSelectQuery string

// ErrEngineVersionSelectNotFound is returned when no Engine Version matches the requested ID.
var ErrEngineVersionSelectNotFound = errors.New("engine version not found")

// EngineVersionSelectRequest identifies an Engine Version for [PgEngineVersionSelect.Exec].
type EngineVersionSelectRequest struct {
	// ID identifies the Engine Version.
	ID uuid.UUID
}

// PgEngineVersionSelect retrieves an immutable Engine Version.
type PgEngineVersionSelect struct{}

// NewPgEngineVersionSelect creates an Engine Version select operation.
func NewPgEngineVersionSelect() *PgEngineVersionSelect {
	return &PgEngineVersionSelect{}
}

// Exec returns the Engine Version or [ErrEngineVersionSelectNotFound].
func (operation *PgEngineVersionSelect) Exec(
	ctx context.Context,
	request *EngineVersionSelectRequest,
) (*EngineVersion, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgEngineVersionSelect")
	defer span.End()

	span.SetAttributes(attribute.String("engine_version.id", request.ID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var engineVersion EngineVersion

	err = db.NewRaw(engineVersionSelectQuery, request.ID).Scan(ctx, &engineVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrEngineVersionSelectNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &engineVersion), nil
}
