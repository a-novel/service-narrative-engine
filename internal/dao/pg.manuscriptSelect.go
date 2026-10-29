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

// ManuscriptRow is one scene of a project with its beat and its cell in the layer a [PgManuscriptSelect]
// reads, or a beat that holds no scene.
type ManuscriptRow struct {
	BeatID uuid.UUID `bun:"beat_id,type:uuid"`
	// BeatTitle names the beat.
	BeatTitle string `bun:"beat_title"`
	// BeatDescription says what the beat is for, written for the author.
	BeatDescription string `bun:"beat_description"`

	// SceneTitle names the scene, nil for a beat that holds no scene.
	SceneTitle *string `bun:"scene_title"`
	// Content is the scene's cell in the layer, nil when nobody wrote it or the beat holds no scene.
	Content json.RawMessage `bun:"content,type:jsonb"`
}

// ManuscriptSelectRequest is the input to [PgManuscriptSelect.Exec].
type ManuscriptSelectRequest struct {
	ProjectID uuid.UUID
	// LayerID is the layer whose cells to read. A layer of another project matches no cell.
	LayerID uuid.UUID
}

// PgManuscriptSelect reads every beat of a project and every scene in them, in narrative order, each scene
// with its cell in one layer.
type PgManuscriptSelect struct{}

func NewPgManuscriptSelect() *PgManuscriptSelect {
	return &PgManuscriptSelect{}
}

func (dao *PgManuscriptSelect) Exec(ctx context.Context, request *ManuscriptSelectRequest) ([]*ManuscriptRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgManuscriptSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*ManuscriptRow

	err = db.NewRaw(manuscriptSelectQuery, request.ProjectID, request.LayerID).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("manuscript.rows", len(entities)))

	return entities, nil
}
