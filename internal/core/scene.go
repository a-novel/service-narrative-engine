package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// Scene is one column of the Scenes Composer: a unit of narrative order. Its fields describe the
// column; what happens in the scene lives in the cells of its layers.
type Scene struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	// Position orders the project's scenes, from 0.
	Position int
	Title    string
	// Color is the scene's color in the composer, as a lowercase #rrggbb hex code.
	Color string
	// Description says what the scene is for, written for the author.
	Description string
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt    string
	UpdatedAt time.Time
}

func newScene(entity *dao.Scene, _ int) *Scene {
	return &Scene{
		ID:          entity.ID,
		ProjectID:   entity.ProjectID,
		Position:    entity.Position,
		Title:       entity.Title,
		Color:       entity.Color,
		Description: entity.Description,
		Prompt:      entity.Prompt,
		UpdatedAt:   entity.UpdatedAt,
	}
}

// sceneAnchor reads the scene another one follows, which must belong to the project.
func sceneAnchor(ctx context.Context, daoSelect SceneCreateDaoSelect, projectID, id uuid.UUID) (*dao.Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.sceneAnchor")
	defer span.End()

	span.SetAttributes(attribute.String("scene.anchor", id.String()))

	anchor, err := daoSelect.Exec(ctx, &dao.SceneSelectRequest{ID: id})
	if errors.Is(err, dao.ErrSceneSelectNotFound) || (err == nil && anchor.ProjectID != projectID) {
		return nil, otel.ReportError(span, fmt.Errorf("%w: unknown anchor %s", ErrInvalidRequest, id))
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select anchor: %w", err))
	}

	return anchor, nil
}
