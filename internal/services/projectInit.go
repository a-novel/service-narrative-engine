package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

type ProjectInsertRepository interface {
	Exec(ctx context.Context, request *dao.ProjectInsertRequest) (*dao.Project, error)
}

type ProjectInsertRepositorySchemaInsert interface {
	Exec(ctx context.Context, request *dao.SchemaInsertRequest) (*dao.Schema, error)
}

type ProjectInsertRepositoryModuleExists interface {
	Exec(ctx context.Context, request *dao.ModuleSelectRequest) (bool, error)
}

type ProjectInitRequest struct {
	Owner    uuid.UUID `validate:"required"`
	Lang     string    `validate:"required,langs"`
	Title    string    `validate:"required,min=1,max=256"`
	Workflow []string  `validate:"required,min=1,max=64,dive,module,max=512"`
}

type ProjectInit struct {
	projectInsertRepository             ProjectInsertRepository
	projectInsertRepositorySchemaInsert ProjectInsertRepositorySchemaInsert
	moduleExistsRepository              ProjectInsertRepositoryModuleExists
}

func NewProjectInit(
	projectInsertRepository ProjectInsertRepository,
	projectInsertRepositorySchemaInsert ProjectInsertRepositorySchemaInsert,
	moduleExistsRepository ProjectInsertRepositoryModuleExists,
) *ProjectInit {
	return &ProjectInit{
		projectInsertRepository:             projectInsertRepository,
		projectInsertRepositorySchemaInsert: projectInsertRepositorySchemaInsert,
		moduleExistsRepository:              moduleExistsRepository,
	}
}

func (service *ProjectInit) Exec(ctx context.Context, request *ProjectInitRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ProjectInit")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// Validate that all modules in the workflow exist.
	for _, module := range request.Workflow {
		decodedModule := lib.DecodeModule(module)

		exists, err := service.moduleExistsRepository.Exec(ctx, &dao.ModuleSelectRequest{
			ID:         decodedModule.Module,
			Namespace:  decodedModule.Namespace,
			Version:    decodedModule.Version,
			Preversion: decodedModule.Preversion,
		})
		if err != nil {
			return nil, otel.ReportError(span, err)
		}

		if !exists {
			return nil, otel.ReportError(span, dao.ErrModuleSelectNotFound)
		}
	}

	var project *dao.Project

	err = postgres.RunInTx(ctx, nil, func(ctx context.Context, tx bun.IDB) error {
		project, err = service.projectInsertRepository.Exec(ctx, &dao.ProjectInsertRequest{
			ID:       uuid.New(),
			Owner:    request.Owner,
			Lang:     request.Lang,
			Title:    request.Title,
			Workflow: request.Workflow,
			Now:      time.Now().UTC(),
		})
		if err != nil {
			return err
		}

		// Init the schemas.
		for _, module := range request.Workflow {
			decodedModule := lib.DecodeModule(module)

			_, err = service.projectInsertRepositorySchemaInsert.Exec(ctx, &dao.SchemaInsertRequest{
				ID:              uuid.New(),
				ProjectID:       project.ID,
				Owner:           &project.Owner,
				ModuleID:        decodedModule.Module,
				ModuleNamespace: decodedModule.Namespace,
				ModuleVersion:   decodedModule.Version,
				Source:          dao.SchemaSourceUser,
				Data:            map[string]any{},
				Now:             time.Now().UTC(),
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, loadProject(project)), nil
}
