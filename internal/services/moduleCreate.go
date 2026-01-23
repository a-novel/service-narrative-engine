package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
	"github.com/a-novel/service-narrative-engine/internal/models"
)

type ModuleCreateRepository interface {
	Exec(ctx context.Context, request *dao.ModuleInsertRequest) (*dao.Module, error)
}

type ModuleCreateRepositoryDelete interface {
	Exec(ctx context.Context, request *dao.ModuleDeleteRequest) (*dao.Module, error)
}

type ModuleCreateRequest struct {
	Module      string            `validate:"required,module,max=512"`
	Description string            `validate:"required,min=32,max=512"`
	Schema      jsonschema.Schema `validate:"required"`
	UI          models.ModuleUi   `validate:"required"`
	Overwrite   bool
}

type ModuleCreate struct {
	moduleInsertRepository ModuleCreateRepository
	moduleDeleteRepository ModuleCreateRepositoryDelete
}

func NewModuleCreate(
	moduleInsertRepository ModuleCreateRepository,
	moduleDeleteRepository ModuleCreateRepositoryDelete,
) *ModuleCreate {
	return &ModuleCreate{
		moduleInsertRepository: moduleInsertRepository,
		moduleDeleteRepository: moduleDeleteRepository,
	}
}

func (service *ModuleCreate) Exec(ctx context.Context, request *ModuleCreateRequest) (*Module, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ModuleCreate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	decodedModule := lib.DecodeModule(request.Module)

	// Check if the jsonSchema is valid.
	resolved, err := request.Schema.Resolve(&jsonschema.ResolveOptions{
		ValidateDefaults: true,
	})
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	var module *dao.Module

	err = postgres.RunInTx(ctx, nil, func(ctx context.Context, tx bun.IDB) error {
		if request.Overwrite {
			_, err = service.moduleDeleteRepository.Exec(ctx, &dao.ModuleDeleteRequest{
				ID:         decodedModule.Module,
				Namespace:  decodedModule.Namespace,
				Version:    decodedModule.Version,
				Preversion: decodedModule.Preversion,
			})

			if err != nil && !errors.Is(err, dao.ErrModuleDeleteNotFound) {
				return err
			}
		}

		module, err = service.moduleInsertRepository.Exec(ctx, &dao.ModuleInsertRequest{
			ID:          decodedModule.Module,
			Namespace:   decodedModule.Namespace,
			Version:     decodedModule.Version,
			Preversion:  decodedModule.Preversion,
			Description: request.Description,
			Schema:      *resolved.Schema(),
			UI:          request.UI,
			Now:         time.Now().UTC(),
		})

		return err
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, loadModule(module)), nil
}
