package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

type SchemaCreateRepository interface {
	Exec(ctx context.Context, request *dao.SchemaInsertRequest) (*dao.Schema, error)
}

type SchemaCreateRepositoryProjectSelect interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

type SchemaCreateRepositoryModuleSelect interface {
	Exec(ctx context.Context, request *dao.ModuleSelectRequest) (*dao.Module, error)
}

type SchemaCreateRequest struct {
	ID        uuid.UUID      `validate:"required"`
	ProjectID uuid.UUID      `validate:"required"`
	UserID    uuid.UUID      `validate:"required"`
	Module    string         `validate:"required,module,max=512"`
	Source    string         `validate:"required,schemaSource,max=64"`
	Data      map[string]any `validate:"required"`
}

type SchemaCreate struct {
	schemaCreateRepository  SchemaCreateRepository
	projectSelectRepository SchemaCreateRepositoryProjectSelect
	moduleSelectRepository  SchemaCreateRepositoryModuleSelect
}

func NewSchemaCreate(
	schemaCreateRepository SchemaCreateRepository,
	projectSelectRepository SchemaCreateRepositoryProjectSelect,
	moduleSelectRepository SchemaCreateRepositoryModuleSelect,
) *SchemaCreate {
	return &SchemaCreate{
		schemaCreateRepository:  schemaCreateRepository,
		projectSelectRepository: projectSelectRepository,
		moduleSelectRepository:  moduleSelectRepository,
	}
}

func (service *SchemaCreate) Exec(ctx context.Context, request *SchemaCreateRequest) (*Schema, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.SchemaCreate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	decodedModule := lib.DecodeModule(request.Module)

	// =================================================================================================================
	// Project validation
	// =================================================================================================================

	project, err := service.projectSelectRepository.Exec(ctx, &dao.ProjectSelectRequest{
		ID: request.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = VerifyProjectOwnership(project, request.UserID)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	// =================================================================================================================
	// Module validation
	// =================================================================================================================

	err = VerifyModule(project, request.Module)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	moduleContent, err := service.moduleSelectRepository.Exec(ctx, &dao.ModuleSelectRequest{
		ID:         decodedModule.Module,
		Namespace:  decodedModule.Namespace,
		Version:    decodedModule.Version,
		Preversion: decodedModule.Preversion,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	// =================================================================================================================
	// Create data.
	// =================================================================================================================

	schema, err := service.schemaCreateRepository.Exec(ctx, &dao.SchemaInsertRequest{
		ID:               request.ID,
		ProjectID:        request.ProjectID,
		Owner:            &request.UserID,
		ModuleID:         moduleContent.ID,
		ModuleNamespace:  moduleContent.Namespace,
		ModuleVersion:    moduleContent.Version,
		ModulePreversion: moduleContent.Preversion,
		Source:           dao.SchemaSource(request.Source),
		Data:             request.Data,
		Now:              time.Now().UTC(),
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, loadSchema(schema)), nil
}
