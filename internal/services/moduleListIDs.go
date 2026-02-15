package services

import (
	"context"
	"errors"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

type ModuleListIDsRepository interface {
	Exec(ctx context.Context, request *dao.ModuleListIDsRequest) ([]string, error)
}

type ModuleListIDsRequest struct {
	Namespace string `validate:"required,max=128,moduleName"`
	Limit     int    `validate:"required,min=1,max=128"`
	Offset    int    `validate:"omitempty,min=0"`
}

type ModuleListIDs struct {
	moduleListIDsRepository ModuleListIDsRepository
}

func NewModuleListIDs(
	moduleListIDsRepository ModuleListIDsRepository,
) *ModuleListIDs {
	return &ModuleListIDs{
		moduleListIDsRepository: moduleListIDsRepository,
	}
}

func (service *ModuleListIDs) Exec(
	ctx context.Context, request *ModuleListIDsRequest,
) ([]string, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ModuleListIDs")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	ids, err := service.moduleListIDsRepository.Exec(ctx, &dao.ModuleListIDsRequest{
		Namespace: request.Namespace,
		Limit:     request.Limit,
		Offset:    request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, ids), nil
}
