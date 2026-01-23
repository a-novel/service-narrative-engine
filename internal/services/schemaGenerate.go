package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

type SchemaGenerateRepository interface {
	Exec(ctx context.Context, request *dao.ModuleGenerateRequest) (map[string]any, error)
}

type SchemaGenerateRepositorySchemaList interface {
	Exec(ctx context.Context, request *dao.SchemaListRequest) ([]*dao.Schema, error)
}

type SchemaGenerateRepositorySchemaInsert interface {
	Exec(ctx context.Context, request *dao.SchemaInsertRequest) (*dao.Schema, error)
}

type SchemaGenerateRepositoryProjectSelect interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

type SchemaGenerateRepositoryModuleSelect interface {
	Exec(ctx context.Context, request *dao.ModuleSelectRequest) (*dao.Module, error)
}

type SchemaGenerateRequest struct {
	ProjectID uuid.UUID `validate:"required"`
	UserID    uuid.UUID `validate:"required"`
	Module    string    `validate:"required,module,max=512"`
	Lang      string    `validate:"required,langs"`
}

type SchemaGenerate struct {
	schemaGenerateRepository SchemaGenerateRepository
	schemaListRepository     SchemaGenerateRepositorySchemaList
	schemaInsertRepository   SchemaGenerateRepositorySchemaInsert
	projectSelectRepository  SchemaGenerateRepositoryProjectSelect
	moduleSelectRepository   SchemaGenerateRepositoryModuleSelect
}

func NewSchemaGenerate(
	schemaGenerateRepository SchemaGenerateRepository,
	schemaListRepository SchemaGenerateRepositorySchemaList,
	schemaInsertRepository SchemaGenerateRepositorySchemaInsert,
	projectSelectRepository SchemaGenerateRepositoryProjectSelect,
	moduleSelectRepository SchemaGenerateRepositoryModuleSelect,
) *SchemaGenerate {
	return &SchemaGenerate{
		schemaGenerateRepository: schemaGenerateRepository,
		schemaListRepository:     schemaListRepository,
		schemaInsertRepository:   schemaInsertRepository,
		projectSelectRepository:  projectSelectRepository,
		moduleSelectRepository:   moduleSelectRepository,
	}
}

func (service *SchemaGenerate) Exec(ctx context.Context, request *SchemaGenerateRequest) (*Schema, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.SchemaGenerate")
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
	// Module preparation.
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

	ok := lib.JSONSchemaLLM(&moduleContent.Schema)
	// Should not happen.
	if !ok {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidData, ErrInvalidRequest))
	}

	moduleSchema, err := moduleContent.Schema.Resolve(&jsonschema.ResolveOptions{
		ValidateDefaults: true,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	moduleContent.Schema = *moduleSchema.Schema()

	// =================================================================================================================
	// Prepare context.
	// =================================================================================================================

	schemas, err := service.schemaListRepository.Exec(ctx, &dao.SchemaListRequest{ProjectID: request.ProjectID})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	// Ignore the module we want to generate.
	contextSchemas := lo.Filter(schemas, func(item *dao.Schema, _ int) bool {
		return item.ModuleNamespace != decodedModule.Namespace || item.ModuleID != decodedModule.Module
	})

	currentSchema, _ := lo.Find(schemas, func(item *dao.Schema) bool {
		return item.ModuleNamespace == decodedModule.Namespace && item.ModuleID == decodedModule.Module
	})

	var prefilled map[string]any
	if currentSchema != nil {
		prefilled = currentSchema.Data
	}

	// =================================================================================================================
	// Generate.
	// =================================================================================================================

	data, err := service.schemaGenerateRepository.Exec(ctx, &dao.ModuleGenerateRequest{
		Module:    moduleContent,
		Lang:      request.Lang,
		Context:   contextSchemas,
		Prefilled: prefilled,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	schema, err := service.schemaInsertRepository.Exec(ctx, &dao.SchemaInsertRequest{
		ID:               uuid.New(),
		ProjectID:        request.ProjectID,
		Owner:            &request.UserID,
		ModuleID:         moduleContent.ID,
		ModuleNamespace:  moduleContent.Namespace,
		ModuleVersion:    moduleContent.Version,
		ModulePreversion: moduleContent.Preversion,
		Source:           dao.SchemaSourceAI,
		Data:             data,
		Now:              time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, loadSchema(schema)), nil
}
