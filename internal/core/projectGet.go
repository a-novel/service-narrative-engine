package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectGetDaoOwner checks that the user owns the project.
type ProjectGetDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// ProjectGetDao reads the project.
type ProjectGetDao interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// ProjectGetRequest names a project of the caller.
type ProjectGetRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
}

// ProjectGet reads a project of the caller. Its beats, scenes, layers and cells come page by page from
// [BeatList], [SceneList], [LayerList] and [CellList].
type ProjectGet struct {
	daoOwner ProjectGetDaoOwner
	dao      ProjectGetDao
}

func NewProjectGet(daoOwner ProjectGetDaoOwner, dao ProjectGetDao) *ProjectGet {
	return &ProjectGet{daoOwner: daoOwner, dao: dao}
}

func (service *ProjectGet) Exec(ctx context.Context, request *ProjectGetRequest) (*Project, error) {
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

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	project, err := service.dao.Exec(ctx, &dao.ProjectSelectRequest{ID: request.ID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select project: %w", err))
	}

	return newProject(project), nil
}
