package services

import (
	"context"
	"errors"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

type ModuleSelectRepository interface {
	Exec(ctx context.Context, request *dao.ModuleSelectRequest) (*dao.Module, error)
}

type ModuleSelectRequest struct {
	// Module is the module string in the format "namespace:module@version".
	Module string `validate:"required,module,max=512"`
}

type ModuleSelect struct {
	moduleSelectRepository ModuleSelectRepository
}

func NewModuleSelect(
	moduleSelectRepository ModuleSelectRepository,
) *ModuleSelect {
	return &ModuleSelect{
		moduleSelectRepository: moduleSelectRepository,
	}
}

func (service *ModuleSelect) Exec(ctx context.Context, request *ModuleSelectRequest) (*Module, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ModuleSelect")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	decodedModule := lib.DecodeModule(request.Module)

	module, err := service.moduleSelectRepository.Exec(ctx, &dao.ModuleSelectRequest{
		ID:         decodedModule.Module,
		Namespace:  decodedModule.Namespace,
		Version:    decodedModule.Version,
		Preversion: decodedModule.Preversion,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, loadModule(module)), nil
}
