package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.cellSummaryList.sql
var cellSummaryListQuery string

// CellSummaryListRequest is the input to [PgCellSummaryList.Exec].
type CellSummaryListRequest struct {
	// ProjectID is the project whose cells to summarize.
	ProjectID uuid.UUID

	// Limit caps the cells in the page.
	Limit int
	// Offset is how many cells come before the page, in the order the page follows.
	Offset int
}

// CellSummaryListRow summarizes one cell of a page: what its documents hold, measured without their text.
type CellSummaryListRow struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID `bun:"scene_id,type:uuid"`
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID `bun:"layer_id,type:uuid"`

	UpdatedAt time.Time `bun:"updated_at"`

	// Empty marks a content that holds no text and no block.
	Empty bool `bun:"empty"`
	// Size counts the characters of every text, speaker and tone the content holds.
	Size int `bun:"size"`
	// Speakers lists the speakers of the content's blocks, each once.
	Speakers []string `bun:"speakers,array"`

	// RecapNames lists the names of the cell's recap. The recap fields are nil until a recap is written.
	RecapNames []string `bun:"recap_names,type:jsonb"`
	// RecapCondensedSize, RecapCompactSize and RecapEssentialsSize count the characters of each version.
	RecapCondensedSize  *int `bun:"recap_condensed_size"`
	RecapCompactSize    *int `bun:"recap_compact_size"`
	RecapEssentialsSize *int `bun:"recap_essentials_size"`
	// RecapCellUpdatedAt is the cell's UpdatedAt the recap read.
	RecapCellUpdatedAt *time.Time `bun:"recap_cell_updated_at,nullzero"`

	// Total counts every cell of the project, beyond the page.
	Total int `bun:"total"`
}

// PgCellSummaryList reads a page of the summaries of a project's cells, by layer, then by scene. Each row
// carries the total, and a page past the end comes back empty.
type PgCellSummaryList struct{}

func NewPgCellSummaryList() *PgCellSummaryList {
	return &PgCellSummaryList{}
}

func (dao *PgCellSummaryList) Exec(
	ctx context.Context, request *CellSummaryListRequest,
) ([]*CellSummaryListRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCellSummaryList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*CellSummaryListRow

	err = db.NewRaw(cellSummaryListQuery, request.ProjectID, request.Limit, request.Offset).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("cells.count", len(entities)))

	return entities, nil
}
