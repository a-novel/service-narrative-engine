package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

type SchemaListVersionsRepository interface {
	Exec(ctx context.Context, request *dao.SchemaListVersionsRequest) ([]*dao.SchemaVersion, error)
}

type SchemaListVersionsRepositoryProjectSelect interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

type SchemaListVersionsRequest struct {
	ProjectID       uuid.UUID `validate:"required"`
	UserID          uuid.UUID `validate:"required"`
	ModuleID        string    `validate:"required,max=128,moduleName"`
	ModuleNamespace string    `validate:"required,max=128,moduleName"`
	Limit           int       `validate:"required,min=1,max=128"`
	Offset          int       `validate:"omitempty,min=0,max=8192"`
}

type SchemaListVersions struct {
	schemaListVersionsRepository SchemaListVersionsRepository
	projectSelectRepository      SchemaListVersionsRepositoryProjectSelect
}

func NewSchemaListVersions(
	schemaListVersionsRepository SchemaListVersionsRepository,
	projectSelectRepository SchemaListVersionsRepositoryProjectSelect,
) *SchemaListVersions {
	return &SchemaListVersions{
		schemaListVersionsRepository: schemaListVersionsRepository,
		projectSelectRepository:      projectSelectRepository,
	}
}

func (service *SchemaListVersions) Exec(
	ctx context.Context, request *SchemaListVersionsRequest,
) ([]*SchemaVersion, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.SchemaListVersions")
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
	// List schema versions
	// =================================================================================================================

	versions, err := service.schemaListVersionsRepository.Exec(ctx, &dao.SchemaListVersionsRequest{
		ProjectID:       request.ProjectID,
		ModuleID:        request.ModuleID,
		ModuleNamespace: request.ModuleNamespace,
		Limit:           request.Limit,
		Offset:          request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, lo.Map(versions, loadSchemaVersionsMap)), nil
}
