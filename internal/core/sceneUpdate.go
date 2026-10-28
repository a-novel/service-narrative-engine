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

// SceneUpdateDaoOwner checks that the user owns the project.
type SceneUpdateDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// SceneUpdateDao writes the scene's new fields.
type SceneUpdateDao interface {
	Exec(ctx context.Context, request *dao.SceneUpdateRequest) (*dao.Scene, error)
}

// SceneUpdateRequest carries a scene's new fields.
type SceneUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// Title names the scene in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color string `validate:"required,len=7,hexcolor"`
	// Description says what the scene is for, written for the author.
	Description string `validate:"max=20000"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string `validate:"max=20000"`
}

// SceneUpdate replaces a scene's title, color, description and prompt.
type SceneUpdate struct {
	daoOwner SceneUpdateDaoOwner
	dao      SceneUpdateDao
}

func NewSceneUpdate(daoOwner SceneUpdateDaoOwner, dao SceneUpdateDao) *SceneUpdate {
	return &SceneUpdate{daoOwner: daoOwner, dao: dao}
}

func (service *SceneUpdate) Exec(ctx context.Context, request *SceneUpdateRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("scene.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	owned, err := service.daoOwner.Exec(ctx, &dao.ProjectOwnedRequest{ID: request.ProjectID, OwnerID: request.OwnerID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	if !owned {
		return nil, otel.ReportError(span, ErrProjectNotFound)
	}

	entity, err := service.dao.Exec(ctx, &dao.SceneUpdateRequest{
		ProjectID:   request.ProjectID,
		ID:          request.ID,
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

	return newScene(entity), nil
}
