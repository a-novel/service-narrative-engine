package core

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectCreateDaoProject inserts the project.
type ProjectCreateDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectInsertRequest) (*dao.Project, error)
}

// ProjectCreateDaoLayers inserts the project's required layers.
type ProjectCreateDaoLayers interface {
	Exec(ctx context.Context, request *dao.LayerInsertRequest) ([]*dao.Layer, error)
}

// ProjectCreateRequest carries a new project.
type ProjectCreateRequest struct {
	OwnerID uuid.UUID `validate:"required"`

	// Title names the narrative.
	Title string `validate:"required,notblank,max=200"`
	// Premise says what the narrative is about, in a few sentences.
	Premise string `validate:"max=20000"`
	// Language is the language the project is written in, one of [ProjectLanguages].
	Language string `validate:"required,language"`
}

// ProjectCreate creates a project with its required layers, in one transaction.
type ProjectCreate struct {
	daoProject ProjectCreateDaoProject
	daoLayers  ProjectCreateDaoLayers
	transactor transaction.Transactor
}

func NewProjectCreate(
	daoProject ProjectCreateDaoProject,
	daoLayers ProjectCreateDaoLayers,
	transactor transaction.Transactor,
) *ProjectCreate {
	return &ProjectCreate{daoProject: daoProject, daoLayers: daoLayers, transactor: transactor}
}

func (service *ProjectCreate) Exec(ctx context.Context, request *ProjectCreateRequest) (*ProjectComposer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectCreate")
	defer span.End()

	span.SetAttributes(attribute.String("project.owner_id", request.OwnerID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// UUIDv7 sorts by creation time, which lets ProjectList order projects by ID. The project takes the
	// first ID, then its required layers.
	ids, err := newIDs(ctx, 1+len(requiredLayers))
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	now := time.Now()
	composer := &ProjectComposer{}

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		project, err := service.daoProject.Exec(ctx, &dao.ProjectInsertRequest{
			ID:       ids[0],
			OwnerID:  request.OwnerID,
			Title:    request.Title,
			Premise:  request.Premise,
			Language: request.Language,
			Now:      now,
		})
		if err != nil {
			return fmt.Errorf("insert project: %w", err)
		}

		kinds := slices.Sorted(maps.Keys(requiredLayers))

		layers, err := service.daoLayers.Exec(ctx, &dao.LayerInsertRequest{
			ProjectID: project.ID,
			Position:  0,
			Layers: lo.Map(kinds, func(kind string, index int) dao.LayerInsertItem {
				return dao.LayerInsertItem{ID: ids[1+index], Kind: kind, Name: requiredLayers[kind]}
			}),
			Now: now,
		})
		if err != nil {
			return fmt.Errorf("insert required layers: %w", err)
		}

		slices.SortFunc(layers, func(a, b *dao.Layer) int { return cmp.Compare(a.Position, b.Position) })

		composer.Project = newProject(project)
		composer.Layers = lo.Map(layers, func(entity *dao.Layer, _ int) *Layer { return newLayer(entity) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return composer, nil
}
