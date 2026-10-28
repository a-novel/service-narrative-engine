package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

const (
	// LayerKindManuscript is the kind of the permanent layer whose cells the manuscript exports.
	LayerKindManuscript = "manuscript"
	// LayerKindNotes is the kind of a layer of free text.
	LayerKindNotes = "notes"
)

// Layer is one row of the Scenes Composer: one kind of information followed across scenes.
type Layer struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	// Position orders the project's layers, from 0.
	Position int
	// Kind names the content schema the layer's cells follow.
	Kind        string
	Name        string
	Description string
	UpdatedAt   time.Time
}

func newLayer(entity *dao.Layer, _ int) *Layer {
	return &Layer{
		ID:          entity.ID,
		ProjectID:   entity.ProjectID,
		Position:    entity.Position,
		Kind:        entity.Kind,
		Name:        entity.Name,
		Description: entity.Description,
		UpdatedAt:   entity.UpdatedAt,
	}
}

// layerAnchor reads the layer another one follows, which must belong to the project.
func layerAnchor(ctx context.Context, daoSelect LayerCreateDaoSelect, projectID, id uuid.UUID) (*dao.Layer, error) {
	anchor, err := daoSelect.Exec(ctx, &dao.LayerSelectRequest{ID: id})
	if errors.Is(err, dao.ErrLayerSelectNotFound) || (err == nil && anchor.ProjectID != projectID) {
		return nil, fmt.Errorf("%w: unknown anchor %s", ErrInvalidRequest, id)
	}

	if err != nil {
		return nil, fmt.Errorf("select anchor: %w", err)
	}

	return anchor, nil
}
