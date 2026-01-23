package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

type ProjectSelectRepository interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

type ProjectSelectRequest struct {
	ID     uuid.UUID `validate:"required"`
	UserID uuid.UUID `validate:"required"`
}

type ProjectSelect struct {
	projectSelectRepository ProjectSelectRepository
}

func NewProjectSelect(
	projectSelectRepository ProjectSelectRepository,
) *ProjectSelect {
	return &ProjectSelect{
		projectSelectRepository: projectSelectRepository,
	}
}

func (service *ProjectSelect) Exec(ctx context.Context, request *ProjectSelectRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ProjectSelect")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	project, err := service.projectSelectRepository.Exec(ctx, &dao.ProjectSelectRequest{
		ID: request.ID,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = VerifyProjectOwnership(project, request.UserID)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, loadProject(project)), nil
}
