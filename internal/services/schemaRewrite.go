package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

type SchemaRewriteRepository interface {
	Exec(ctx context.Context, request *dao.SchemaUpdateRequest) (*dao.Schema, error)
}

type SchemaRewriteRepositoryProjectSelect interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

type SchemaRewriteRepositorySchemaSelect interface {
	Exec(ctx context.Context, request *dao.SchemaSelectRequest) (*dao.Schema, error)
}

type SchemaRewriteRequest struct {
	ID     uuid.UUID      `validate:"required"`
	UserID uuid.UUID      `validate:"required"`
	Data   map[string]any `validate:"required"`
	Now    time.Time
}

type SchemaRewrite struct {
	schemaRewriteRepository SchemaRewriteRepository
	projectSelectRepository SchemaRewriteRepositoryProjectSelect
	schemaSelectRepository  SchemaRewriteRepositorySchemaSelect
}

func NewSchemaRewrite(
	schemaRewriteRepository SchemaRewriteRepository,
	projectSelectRepository SchemaRewriteRepositoryProjectSelect,
	schemaSelectRepository SchemaRewriteRepositorySchemaSelect,
) *SchemaRewrite {
	return &SchemaRewrite{
		schemaRewriteRepository: schemaRewriteRepository,
		projectSelectRepository: projectSelectRepository,
		schemaSelectRepository:  schemaSelectRepository,
	}
}

func (service *SchemaRewrite) Exec(ctx context.Context, request *SchemaRewriteRequest) (*Schema, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.SchemaRewrite")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	currentSchema, err := service.schemaSelectRepository.Exec(ctx, &dao.SchemaSelectRequest{
		ID: &request.ID,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	// Schemas with `null` content are special history entries indicating a module was removed from a workflow.
	// Those cannot be rewritten. If a user still attempts to rewrite such an entry (which is not hidden from the
	// client), return a 'not found' error. The client-side code should prevent such events.
	if currentSchema.Data == nil {
		return nil, otel.ReportError(span, dao.ErrSchemaSelectNotFound)
	}

	// =================================================================================================================
	// Project validation
	// =================================================================================================================

	project, err := service.projectSelectRepository.Exec(ctx, &dao.ProjectSelectRequest{
		ID: currentSchema.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = VerifyProjectOwnership(project, request.UserID)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	// =================================================================================================================
	// Rewrite data.
	// =================================================================================================================

	schema, err := service.schemaRewriteRepository.Exec(ctx, &dao.SchemaUpdateRequest{
		ID:   request.ID,
		Data: request.Data,
		Now:  request.Now,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, loadSchema(schema)), nil
}
