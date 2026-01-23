package services

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/modules"
)

type ModuleLoadSystemRepository interface {
	Exec(ctx context.Context, request *dao.ModuleInsertRequest) (*dao.Module, error)
}

type ModuleLoadSystemRepositoryDelete interface {
	Exec(ctx context.Context, request *dao.ModuleDeleteRequest) (*dao.Module, error)
}

type ModuleLoadSystemRepositorySelect interface {
	Exec(ctx context.Context, request *dao.ModuleSelectRequest) (*dao.Module, error)
}

type ModuleLoadSystemRepositoryListVersions interface {
	Exec(ctx context.Context, request *dao.ModuleListVersionsRequest) ([]*dao.ModuleVersion, error)
}

type ModuleLoadSystemRequest struct {
	Module  modules.SystemModule
	Version string `validate:"required,moduleVersion"`
	DevMode bool
}

type ModuleLoadSystem struct {
	moduleLoadSystemRepository             ModuleLoadSystemRepository
	moduleLoadSystemRepositoryDelete       ModuleLoadSystemRepositoryDelete
	moduleLoadSystemRepositorySelect       ModuleLoadSystemRepositorySelect
	moduleLoadSystemRepositoryListVersions ModuleLoadSystemRepositoryListVersions
}

func NewModuleLoadSystem(
	moduleLoadSystemRepository ModuleLoadSystemRepository,
	moduleLoadSystemRepositoryDelete ModuleLoadSystemRepositoryDelete,
	moduleLoadSystemRepositorySelect ModuleLoadSystemRepositorySelect,
	moduleLoadSystemRepositoryListVersions ModuleLoadSystemRepositoryListVersions,
) *ModuleLoadSystem {
	return &ModuleLoadSystem{
		moduleLoadSystemRepository:             moduleLoadSystemRepository,
		moduleLoadSystemRepositoryDelete:       moduleLoadSystemRepositoryDelete,
		moduleLoadSystemRepositorySelect:       moduleLoadSystemRepositorySelect,
		moduleLoadSystemRepositoryListVersions: moduleLoadSystemRepositoryListVersions,
	}
}

// Exec loads the system modules provided through the embedded file system.
//
// System modules are treated differently from user modules. First, their version is directly tied to the service
// itself, so new versions are only published as part of new deployments.
//
// For experimentation during local development, however, stable versions are never published. Instead, a new
// pre-version is created over the current deployment version each time the service is started and new changes are
// detected.
func (service *ModuleLoadSystem) Exec(
	ctx context.Context, request *ModuleLoadSystemRequest,
) (*Module, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ModuleLoadSystem")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if request.DevMode {
		// Retrieve the latest module for the current version, to verify it there are any changes.
		versionsList, err := service.moduleLoadSystemRepositoryListVersions.Exec(ctx, &dao.ModuleListVersionsRequest{
			ID:         request.Module.ID,
			Namespace:  request.Module.Namespace,
			Limit:      1, // Select only the latest iteration.
			Version:    request.Version,
			Preversion: true,
		})
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("failed to list module versions: %w", err))
		}

		if len(versionsList) > 0 {
			latest, err := service.moduleLoadSystemRepositorySelect.Exec(ctx, &dao.ModuleSelectRequest{
				ID:         request.Module.ID,
				Namespace:  request.Module.Namespace,
				Version:    versionsList[0].Version,
				Preversion: versionsList[0].Preversion,
			})
			if err != nil {
				return nil, otel.ReportError(span, fmt.Errorf("failed to retrieve latest module: %w", err))
			}

			if reflect.DeepEqual(latest.Schema, request.Module.Schema) &&
				reflect.DeepEqual(latest.UI, request.Module.UI) {
				return otel.ReportSuccess(span, loadModule(latest)), nil
			}
		}
	}

	// Default mode. Create a new version for the current deployment.
	// In dev mode, prefix UUID with hyphen to match ModulePreversionRegex pattern (-[a-z0-9]+)*
	result, err := service.moduleLoadSystemRepository.Exec(ctx, &dao.ModuleInsertRequest{
		ID:          request.Module.ID,
		Namespace:   request.Module.Namespace,
		Version:     request.Version,
		Preversion:  lo.Ternary(request.DevMode, "-"+uuid.NewString(), ""),
		Description: request.Module.Description,
		Schema:      request.Module.Schema,
		UI:          request.Module.UI,
		Now:         time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("failed to insert module: %w", err))
	}

	return otel.ReportSuccess(span, loadModule(result)), nil
}
