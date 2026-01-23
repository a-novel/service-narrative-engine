package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

type SchemaSelectRepository interface {
	Exec(ctx context.Context, request *dao.SchemaSelectRequest) (*dao.Schema, error)
}

type SchemaSelectRepositoryProjectSelect interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

type SchemaSelectRequest struct {
	ID        *uuid.UUID `validate:"required_without=ProjectID"`
	ProjectID uuid.UUID  `validate:"required_without=ID"`
	Module    string     `validate:"required_without=ID,omitempty,module,max=512"`
	UserID    uuid.UUID  `validate:"required"`
}

type SchemaSelect struct {
	schemaSelectRepository  SchemaSelectRepository
	projectSelectRepository SchemaSelectRepositoryProjectSelect
}

func NewSchemaSelect(
	schemaSelectRepository SchemaSelectRepository,
	projectSelectRepository SchemaSelectRepositoryProjectSelect,
) *SchemaSelect {
	return &SchemaSelect{
		schemaSelectRepository:  schemaSelectRepository,
		projectSelectRepository: projectSelectRepository,
	}
}

func (service *SchemaSelect) Exec(ctx context.Context, request *SchemaSelectRequest) (*Schema, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.SchemaSelect")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

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
	// Fetch schema
	// =================================================================================================================

	daoRequest := &dao.SchemaSelectRequest{
		ID:        request.ID,
		ProjectID: request.ProjectID,
	}

	if request.Module != "" {
		decodedModule := lib.DecodeModule(request.Module)
		daoRequest.ModuleID = decodedModule.Module
		daoRequest.ModuleNamespace = decodedModule.Namespace
	}

	schema, err := service.schemaSelectRepository.Exec(ctx, daoRequest)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, loadSchema(schema)), nil
}
