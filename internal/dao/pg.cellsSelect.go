package dao

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.cellsSelect.sql
var cellsSelectQuery string

// CellsSelectKey names one cell a [PgCellsSelect] reads, and which of its documents.
type CellsSelectKey struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID

	// Content asks for the cell's content.
	Content bool
	// Recap names the version of the cell's recap to read: condensed, compact or essentials. Empty reads none.
	Recap string
}

// CellsSelectRequest is the input to [PgCellsSelect.Exec].
type CellsSelectRequest struct {
	ProjectID uuid.UUID
	// Cells lists the cells to read.
	Cells []*CellsSelectKey
}

// CellsSelectRow is one cell a [PgCellsSelect] read, with the documents its key asked for.
type CellsSelectRow struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID `bun:"scene_id,type:uuid"`
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID `bun:"layer_id,type:uuid"`

	// Content is the cell's content, nil unless its key asked for it.
	Content json.RawMessage `bun:"content,type:jsonb"`
	// RecapVersion is the version of the recap the key asked for, and Recap its text, nil unless the key asked
	// for one or until a recap is written.
	RecapVersion string  `bun:"recap_version"`
	Recap        *string `bun:"recap"`
}

// PgCellsSelect reads cells of a project, in no particular order. It skips a key the project holds no cell
// for.
type PgCellsSelect struct{}

func NewPgCellsSelect() *PgCellsSelect {
	return &PgCellsSelect{}
}

func (dao *PgCellsSelect) Exec(ctx context.Context, request *CellsSelectRequest) ([]*CellsSelectRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCellsSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	scenes := lo.Map(request.Cells, func(key *CellsSelectKey, _ int) uuid.UUID { return key.SceneID })
	layers := lo.Map(request.Cells, func(key *CellsSelectKey, _ int) uuid.UUID { return key.LayerID })
	contents := lo.Map(request.Cells, func(key *CellsSelectKey, _ int) bool { return key.Content })
	recaps := lo.Map(request.Cells, func(key *CellsSelectKey, _ int) string { return key.Recap })

	var entities []*CellsSelectRow

	err = db.NewRaw(
		cellsSelectQuery, request.ProjectID, bun.List(scenes), bun.List(layers), bun.List(contents), bun.List(recaps),
		len(request.Cells),
	).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("cells.count", len(entities)))

	return entities, nil
}
