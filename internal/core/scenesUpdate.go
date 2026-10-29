package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ScenesUpdateDaoOwner checks that the user owns the project.
type ScenesUpdateDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// ScenesUpdateDao writes the scenes' new fields, all of them or none.
type ScenesUpdateDao interface {
	Exec(ctx context.Context, request *dao.ScenesUpdateRequest) ([]*dao.Scene, error)
}

// ScenesUpdateItem is one scene of a [ScenesUpdateRequest].
type ScenesUpdateItem struct {
	ID uuid.UUID `validate:"required"`

	// Title names the scene in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Description refines the beat's description for the scene. Empty keeps the beat's.
	Description string `validate:"max=20000"`
	// Prompt refines the beat's prompt for the scene. Empty keeps the beat's.
	Prompt string `validate:"max=20000"`
}

// ScenesUpdateRequest carries the new fields of scenes.
type ScenesUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// Scenes lists the scenes to update, each once, at most 100.
	Scenes []ScenesUpdateItem `validate:"required,min=1,max=100,unique=ID,dive"`
}

// ScenesUpdate replaces the fields of several scenes at once, all of them or none, and returns them in the
// order of the request. [SceneUpdate] updates one.
type ScenesUpdate struct {
	daoOwner ScenesUpdateDaoOwner
	dao      ScenesUpdateDao
}

func NewScenesUpdate(daoOwner ScenesUpdateDaoOwner, dao ScenesUpdateDao) *ScenesUpdate {
	return &ScenesUpdate{daoOwner: daoOwner, dao: dao}
}

func (service *ScenesUpdate) Exec(ctx context.Context, request *ScenesUpdateRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ScenesUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("scenes.count", len(request.Scenes)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	updated, err := service.dao.Exec(ctx, &dao.ScenesUpdateRequest{
		ProjectID: request.ProjectID,
		Scenes: lo.Map(request.Scenes, func(item ScenesUpdateItem, _ int) dao.ScenesUpdateItem {
			return dao.ScenesUpdateItem{
				ID: item.ID, Title: item.Title, Description: item.Description, Prompt: item.Prompt,
			}
		}),
		Now: time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update scenes: %w", err))
	}

	// The update writes none of the scenes when the project does not hold one of them.
	if len(updated) != len(request.Scenes) {
		return nil, otel.ReportError(span, fmt.Errorf(
			"%w: %d of %d scenes", ErrSceneNotFound, len(request.Scenes)-len(updated), len(request.Scenes),
		))
	}

	byID := lo.KeyBy(updated, func(entity *dao.Scene) uuid.UUID { return entity.ID })

	return lo.Map(request.Scenes, func(item ScenesUpdateItem, _ int) *Scene { return newScene(byID[item.ID]) }), nil
}
