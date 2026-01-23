package services

import (
	"context"
	"errors"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

type ModuleListVersionsRepository interface {
	Exec(ctx context.Context, request *dao.ModuleListVersionsRequest) ([]*dao.ModuleVersion, error)
}

type ModuleListVersionsRequest struct {
	ID        string `validate:"required,max=128,moduleName"`
	Namespace string `validate:"required,max=128,moduleName"`
	Limit     int    `validate:"required,min=1,max=128"`
	Offset    int    `validate:"omitempty,min=0,max=8192"`
	Version   string `validate:"omitempty,moduleVersion,max=32"`
	// Preversion indicates whether to include preversions in the results.
	// By default, only stable versions (empty preversion) are returned.
	Preversion bool
}

type ModuleListVersions struct {
	moduleListVersionsRepository ModuleListVersionsRepository
}

func NewModuleListVersions(
	moduleListVersionsRepository ModuleListVersionsRepository,
) *ModuleListVersions {
	return &ModuleListVersions{
		moduleListVersionsRepository: moduleListVersionsRepository,
	}
}

func (service *ModuleListVersions) Exec(
	ctx context.Context, request *ModuleListVersionsRequest,
) ([]*ModuleVersion, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ModuleListVersions")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	versions, err := service.moduleListVersionsRepository.Exec(ctx, &dao.ModuleListVersionsRequest{
		ID:         request.ID,
		Namespace:  request.Namespace,
		Limit:      request.Limit,
		Offset:     request.Offset,
		Version:    request.Version,
		Preversion: request.Preversion,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, lo.Map(versions, loadModuleVersionsMap)), nil
}
