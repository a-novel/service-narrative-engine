package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

type ProjectDeleteRepositorySelect interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

type ProjectDeleteRepository interface {
	Exec(ctx context.Context, request *dao.ProjectDeleteRequest) (*dao.Project, error)
}

type ProjectDeleteRequest struct {
	ID     uuid.UUID `validate:"required"`
	UserID uuid.UUID `validate:"required"`
}

type ProjectDelete struct {
	projectDeleteRepositorySelect ProjectDeleteRepositorySelect
	projectDeleteRepository       ProjectDeleteRepository
}

func NewProjectDelete(
	projectDeleteRepository ProjectDeleteRepository,
	projectDeleteRepositorySelect ProjectDeleteRepositorySelect,
) *ProjectDelete {
	return &ProjectDelete{
		projectDeleteRepositorySelect: projectDeleteRepositorySelect,
		projectDeleteRepository:       projectDeleteRepository,
	}
}

func (service *ProjectDelete) Exec(ctx context.Context, request *ProjectDeleteRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ProjectDelete")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	project, err := service.projectDeleteRepositorySelect.Exec(ctx, &dao.ProjectSelectRequest{
		ID: request.ID,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = VerifyProjectOwnership(project, request.UserID)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	deletedProject, err := service.projectDeleteRepository.Exec(ctx, &dao.ProjectDeleteRequest{
		ID: request.ID,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, loadProject(deletedProject)), nil
}
