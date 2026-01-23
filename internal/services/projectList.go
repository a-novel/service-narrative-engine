package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

type ProjectListRepository interface {
	Exec(ctx context.Context, request *dao.ProjectListRequest) ([]*dao.Project, error)
}

type ProjectListRequest struct {
	UserID uuid.UUID `validate:"required"`
	Limit  int       `validate:"required,min=1,max=128"`
	Offset int       `validate:"omitempty,min=0,max=8192"`
}

type ProjectList struct {
	projectListRepository ProjectListRepository
}

func NewProjectList(
	projectListRepository ProjectListRepository,
) *ProjectList {
	return &ProjectList{
		projectListRepository: projectListRepository,
	}
}

func (service *ProjectList) Exec(ctx context.Context, request *ProjectListRequest) ([]*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ProjectList")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	projects, err := service.projectListRepository.Exec(ctx, &dao.ProjectListRequest{
		Owner:  request.UserID,
		Limit:  request.Limit,
		Offset: request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, lo.Map(projects, loadProjectsMap)), nil
}
