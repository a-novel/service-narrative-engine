package dao

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.manuscriptSelect.sql
var manuscriptSelectQuery string

// ManuscriptScene is one scene of a project, with its cell in the layer a [PgManuscriptSelect] reads.
type ManuscriptScene struct {
	// Title names the scene.
	Title string `bun:"title"`

	// Content is the scene's cell in the layer, nil when nobody wrote it.
	Content json.RawMessage `bun:"content,type:jsonb"`
}

// ManuscriptSelectRequest is the input to [PgManuscriptSelect.Exec].
type ManuscriptSelectRequest struct {
	ProjectID uuid.UUID
	// LayerID is the layer whose cells to read. A layer of another project matches no cell.
	LayerID uuid.UUID
}

// PgManuscriptSelect reads every scene of a project in narrative order, each with its cell in one layer.
type PgManuscriptSelect struct{}

func NewPgManuscriptSelect() *PgManuscriptSelect {
	return &PgManuscriptSelect{}
}

func (dao *PgManuscriptSelect) Exec(ctx context.Context, request *ManuscriptSelectRequest) ([]*ManuscriptScene, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgManuscriptSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*ManuscriptScene

	err = db.NewRaw(manuscriptSelectQuery, request.ProjectID, request.LayerID).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("scenes.count", len(entities)))

	return entities, nil
}
