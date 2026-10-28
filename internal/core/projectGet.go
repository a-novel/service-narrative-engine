package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectGetDaoProject reads the project.
type ProjectGetDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// ProjectGetDaoScenes reads the project's scenes.
type ProjectGetDaoScenes interface {
	Exec(ctx context.Context, request *dao.SceneListRequest) ([]*dao.Scene, error)
}

// ProjectGetDaoLayers reads the project's layers.
type ProjectGetDaoLayers interface {
	Exec(ctx context.Context, request *dao.LayerListRequest) ([]*dao.Layer, error)
}

// ProjectGetRequest names a project of the caller.
type ProjectGetRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
}

// ProjectGet reads a project with the layout of its Scenes Composer, so a client loads it in one
// call before paging through its cells with [CellList].
type ProjectGet struct {
	daoProject ProjectGetDaoProject
	daoScenes  ProjectGetDaoScenes
	daoLayers  ProjectGetDaoLayers
}

func NewProjectGet(
	daoProject ProjectGetDaoProject,
	daoScenes ProjectGetDaoScenes,
	daoLayers ProjectGetDaoLayers,
) *ProjectGet {
	return &ProjectGet{
		daoProject: daoProject,
		daoScenes:  daoScenes,
		daoLayers:  daoLayers,
	}
}

func (service *ProjectGet) Exec(ctx context.Context, request *ProjectGetRequest) (*ProjectComposer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectGet")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	project, err := service.daoProject.Exec(ctx, &dao.ProjectSelectRequest{ID: request.ID, OwnerID: request.OwnerID})
	if errors.Is(err, dao.ErrProjectSelectNotFound) {
		err = errors.Join(err, ErrProjectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select project: %w", err))
	}

	scenes, err := service.daoScenes.Exec(ctx, &dao.SceneListRequest{ProjectID: project.ID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list scenes: %w", err))
	}

	layers, err := service.daoLayers.Exec(ctx, &dao.LayerListRequest{ProjectID: project.ID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list layers: %w", err))
	}

	return &ProjectComposer{
		Project: newProject(project),
		Scenes:  lo.Map(scenes, newScene),
		Layers:  lo.Map(layers, newLayer),
	}, nil
}
