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

//nolint:lll
var ErrForbiddenModuleUpgrade = errors.New("forbidden module upgrade: you must create a new schema with the updated module version, and provide compatible data")

type ProjectUpdateRepositorySelect interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

type ProjectUpdateRepository interface {
	Exec(ctx context.Context, request *dao.ProjectUpdateRequest) (*dao.Project, error)
}

type ProjectUpdateRepositorySchemaInsert interface {
	Exec(ctx context.Context, request *dao.SchemaInsertRequest) (*dao.Schema, error)
}

type ProjectUpdateRepositoryModuleExists interface {
	Exec(ctx context.Context, request *dao.ModuleSelectRequest) (bool, error)
}

type ProjectUpdateRequest struct {
	ID       uuid.UUID `validate:"required"`
	UserID   uuid.UUID `validate:"required"`
	Workflow []string  `validate:"required,min=1,max=64,dive,module,max=512"`
	Title    string    `validate:"required,min=1,max=256"`
}

type ProjectUpdate struct {
	projectUpdateRepositorySelect       ProjectUpdateRepositorySelect
	projectUpdateRepository             ProjectUpdateRepository
	projectUpdateRepositorySchemaInsert ProjectUpdateRepositorySchemaInsert
	moduleExistsRepository              ProjectUpdateRepositoryModuleExists
}

func NewProjectUpdate(
	projectUpdateRepository ProjectUpdateRepository,
	projectUpdateRepositorySelect ProjectUpdateRepositorySelect,
	projectUpdateRepositorySchemaInsert ProjectUpdateRepositorySchemaInsert,
	moduleExistsRepository ProjectUpdateRepositoryModuleExists,
) *ProjectUpdate {
	return &ProjectUpdate{
		projectUpdateRepositorySelect:       projectUpdateRepositorySelect,
		projectUpdateRepository:             projectUpdateRepository,
		projectUpdateRepositorySchemaInsert: projectUpdateRepositorySchemaInsert,
		moduleExistsRepository:              moduleExistsRepository,
	}
}

func (service *ProjectUpdate) Exec(ctx context.Context, request *ProjectUpdateRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ProjectUpdate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	project, err := service.projectUpdateRepositorySelect.Exec(ctx, &dao.ProjectSelectRequest{
		ID: request.ID,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = VerifyProjectOwnership(project, request.UserID)
	if err != nil {
		return nil, otel.ReportError(span, err)
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

	// Check for changes in the project workflow and update schemas accordingly.
	projectModules := map[string]string{}
	for _, module := range project.Workflow {
		projectModules[lib.VersionlessModule(module)] = module
	}

	requestModules := map[string]string{}
	for _, module := range request.Workflow {
		requestModules[lib.VersionlessModule(module)] = module
	}

	var addedModules, removedModules []string

	for rModule, rFullModule := range requestModules {
		if fullModule, exists := projectModules[rModule]; !exists {
			addedModules = append(addedModules, rFullModule)
		} else if fullModule != rFullModule {
			// Make sure no module has been upgraded, as this operation should be performed using its own update
			// mechanism.
			return nil, ErrForbiddenModuleUpgrade
		}
	}

	for pModule, pVersion := range projectModules {
		if _, exists := requestModules[pModule]; !exists {
			removedModules = append(removedModules, pVersion)
		}
	}

	var updatedProject *dao.Project

	err = postgres.RunInTx(ctx, nil, func(ctx context.Context, tx bun.IDB) error {
		updatedProject, err = service.projectUpdateRepository.Exec(ctx, &dao.ProjectUpdateRequest{
			ID:       request.ID,
			Title:    request.Title,
			Workflow: request.Workflow,
			Now:      time.Now().UTC(),
		})
		if err != nil {
			return err
		}

		for _, module := range addedModules {
			decodedModule := lib.DecodeModule(module)

			_, err = service.projectUpdateRepositorySchemaInsert.Exec(ctx, &dao.SchemaInsertRequest{
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

		for _, module := range removedModules {
			decodedModule := lib.DecodeModule(module)

			_, err = service.projectUpdateRepositorySchemaInsert.Exec(ctx, &dao.SchemaInsertRequest{
				ID:              uuid.New(),
				ProjectID:       project.ID,
				Owner:           &project.Owner,
				ModuleID:        decodedModule.Module,
				ModuleNamespace: decodedModule.Namespace,
				ModuleVersion:   decodedModule.Version,
				Source:          dao.SchemaSourceUser,
				Data:            nil,
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

	return otel.ReportSuccess(span, loadProject(updatedProject)), nil
}
