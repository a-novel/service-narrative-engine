package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.layerKind.sql
var layerKindQuery string

// ErrLayerKindNotFound is returned by [PgLayerKind.Exec] when the project holds no layer with the ID.
var ErrLayerKindNotFound = errors.New("layer not found")

// LayerKindRequest is the input to [PgLayerKind.Exec].
type LayerKindRequest struct {
	ProjectID uuid.UUID
	ID        uuid.UUID
}

// PgLayerKind reads the kind of a layer of a project, which names the content schema of its cells.
type PgLayerKind struct{}

func NewPgLayerKind() *PgLayerKind {
	return &PgLayerKind{}
}

func (dao *PgLayerKind) Exec(ctx context.Context, request *LayerKindRequest) (string, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgLayerKind")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return "", otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var kind string

	err = db.NewRaw(layerKindQuery, request.ProjectID, request.ID).Scan(ctx, &kind)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrLayerKindNotFound)
	}

	if err != nil {
		return "", otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return kind, nil
}
