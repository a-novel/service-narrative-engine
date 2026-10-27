package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// SceneUpdateDao writes the scene's new fields.
type SceneUpdateDao interface {
	Exec(ctx context.Context, request *dao.SceneUpdateRequest) (*dao.Scene, error)
}

// SceneUpdateRequest carries a scene's new fields.
type SceneUpdateRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
	Title   string    `validate:"required,notblank,max=200"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color       string `validate:"required,len=7,hexcolor"`
	Description string `validate:"max=20000"`
	Prompt      string `validate:"max=20000"`
}

// SceneUpdate replaces a scene's title, color, description and prompt.
type SceneUpdate struct {
	dao SceneUpdateDao
}

func NewSceneUpdate(dao SceneUpdateDao) *SceneUpdate {
	return &SceneUpdate{dao: dao}
}

func (service *SceneUpdate) Exec(ctx context.Context, request *SceneUpdateRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("scene.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.SceneUpdateRequest{
		ID:          request.ID,
		OwnerID:     request.OwnerID,
		Title:       request.Title,
		Color:       strings.ToLower(request.Color),
		Description: request.Description,
		Prompt:      request.Prompt,
		Now:         time.Now(),
	})
	if errors.Is(err, dao.ErrSceneUpdateNotFound) {
		err = errors.Join(err, ErrSceneNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update scene: %w", err))
	}

	return newScene(entity, 0), nil
}
