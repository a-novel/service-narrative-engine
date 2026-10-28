package core

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
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

// ProjectCreateDaoLayer inserts the project's required layers.
type ProjectCreateDaoLayer interface {
	Exec(ctx context.Context, request *dao.LayerInsertRequest) (*dao.Layer, error)
}

// ProjectCreateDaoScenes inserts the scenes the project starts with.
type ProjectCreateDaoScenes interface {
	Exec(ctx context.Context, request *dao.SceneInsertRequest) ([]*dao.Scene, error)
}

// ProjectCreateScene is a scene a new project starts with, such as one a preset seeds.
type ProjectCreateScene struct {
	// Title names the scene in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color string `validate:"required,len=7,hexcolor"`
	// Description says what the scene is for, written for the author.
	Description string `validate:"max=20000"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string `validate:"max=20000"`
}

// ProjectCreateRequest carries a new project and the scenes it starts with.
type ProjectCreateRequest struct {
	OwnerID uuid.UUID `validate:"required"`

	// Title names the narrative.
	Title string `validate:"required,notblank,max=200"`
	// Premise says what the narrative is about, in a few sentences.
	Premise string `validate:"max=20000"`
	// Language is the language the project is written in.
	Language string `validate:"required,oneof=en fr"`

	// Scenes are the scenes the project starts with, in order, up to the 500 a project holds.
	Scenes []ProjectCreateScene `validate:"max=500,dive"`
}

// ProjectCreate creates a project with its required layers and its first scenes, in one
// transaction.
type ProjectCreate struct {
	daoProject ProjectCreateDaoProject
	daoLayer   ProjectCreateDaoLayer
	daoScenes  ProjectCreateDaoScenes
	transactor transaction.Transactor
}

func NewProjectCreate(
	daoProject ProjectCreateDaoProject,
	daoLayer ProjectCreateDaoLayer,
	daoScenes ProjectCreateDaoScenes,
	transactor transaction.Transactor,
) *ProjectCreate {
	return &ProjectCreate{
		daoProject: daoProject,
		daoLayer:   daoLayer,
		daoScenes:  daoScenes,
		transactor: transactor,
	}
}

func (service *ProjectCreate) Exec(ctx context.Context, request *ProjectCreateRequest) (*ProjectComposer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("scenes.count", len(request.Scenes)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// UUIDv7 sorts by creation time, which lets ProjectList order projects by ID. The project takes the
	// first ID, then its required layers, then its scenes.
	ids, err := newIDs(1 + len(requiredLayers) + len(request.Scenes))
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	layerIDs, sceneIDs := ids[1:1+len(requiredLayers)], ids[1+len(requiredLayers):]

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

		composer.Project = newProject(project)

		for position, kind := range slices.Sorted(maps.Keys(requiredLayers)) {
			layer, err := service.daoLayer.Exec(ctx, &dao.LayerInsertRequest{
				ID:        layerIDs[position],
				ProjectID: project.ID,
				Position:  position,
				Kind:      kind,
				Name:      requiredLayers[kind],
				Now:       now,
			})
			if err != nil {
				return fmt.Errorf("insert %s layer: %w", kind, err)
			}

			composer.Layers = append(composer.Layers, newLayer(layer))
		}

		if len(request.Scenes) == 0 {
			return nil
		}

		scenes, err := service.daoScenes.Exec(ctx, &dao.SceneInsertRequest{
			ProjectID: project.ID,
			Position:  0,
			Scenes:    lo.Map(request.Scenes, seedScene(sceneIDs)),
			Now:       now,
		})
		if err != nil {
			return fmt.Errorf("insert scenes: %w", err)
		}

		slices.SortFunc(scenes, func(a, b *dao.Scene) int { return cmp.Compare(a.Position, b.Position) })
		composer.Scenes = lo.Map(scenes, func(entity *dao.Scene, _ int) *Scene { return newScene(entity) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return composer, nil
}

// seedScene adapts the scenes of a request to lo.Map, giving each the ID at its index.
func seedScene(ids []uuid.UUID) func(scene ProjectCreateScene, index int) dao.SceneInsertItem {
	return func(scene ProjectCreateScene, index int) dao.SceneInsertItem {
		return dao.SceneInsertItem{
			ID: ids[index], Title: scene.Title, Color: strings.ToLower(scene.Color),
			Description: scene.Description, Prompt: scene.Prompt,
		}
	}
}
