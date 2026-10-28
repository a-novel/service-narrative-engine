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

// SceneUpdateDaoSelect reads the scene, to find its project.
type SceneUpdateDaoSelect interface {
	Exec(ctx context.Context, request *dao.SceneSelectRequest) (*dao.Scene, error)
}

// SceneUpdateDaoProject reads the scene's project, to check its owner.
type SceneUpdateDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

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
	daoSelect  SceneUpdateDaoSelect
	daoProject SceneUpdateDaoProject
	dao        SceneUpdateDao
}

func NewSceneUpdate(daoSelect SceneUpdateDaoSelect, daoProject SceneUpdateDaoProject, dao SceneUpdateDao) *SceneUpdate {
	return &SceneUpdate{daoSelect: daoSelect, daoProject: daoProject, dao: dao}
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

	target, err := service.daoSelect.Exec(ctx, &dao.SceneSelectRequest{ID: request.ID})
	if errors.Is(err, dao.ErrSceneSelectNotFound) {
		err = errors.Join(err, ErrSceneNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select scene: %w", err))
	}

	project, err := service.daoProject.Exec(ctx, &dao.ProjectSelectRequest{ID: target.ProjectID})
	if errors.Is(err, dao.ErrProjectSelectNotFound) {
		err = errors.Join(err, ErrSceneNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select project: %w", err))
	}

	err = checkOwner(project, request.OwnerID, ErrSceneNotFound)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	entity, err := service.dao.Exec(ctx, &dao.SceneUpdateRequest{
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

	return newScene(entity, 0), nil
}
