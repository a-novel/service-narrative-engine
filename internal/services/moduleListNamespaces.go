package services

import (
	"context"
	"errors"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

type ModuleListNamespacesRepository interface {
	Exec(ctx context.Context, request *dao.ModuleListNamespacesRequest) ([]string, error)
}

type ModuleListNamespacesRequest struct {
	Limit  int `validate:"required,min=1,max=128"`
	Offset int `validate:"omitempty,min=0"`
}

type ModuleListNamespaces struct {
	moduleListNamespacesRepository ModuleListNamespacesRepository
}

func NewModuleListNamespaces(
	moduleListNamespacesRepository ModuleListNamespacesRepository,
) *ModuleListNamespaces {
	return &ModuleListNamespaces{
		moduleListNamespacesRepository: moduleListNamespacesRepository,
	}
}

func (service *ModuleListNamespaces) Exec(
	ctx context.Context, request *ModuleListNamespacesRequest,
) ([]string, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ModuleListNamespaces")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	namespaces, err := service.moduleListNamespacesRepository.Exec(ctx, &dao.ModuleListNamespacesRequest{
		Limit:  request.Limit,
		Offset: request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, namespaces), nil
}
