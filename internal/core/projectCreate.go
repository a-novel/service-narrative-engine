package core

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// manuscriptLayerName names the manuscript layer of a new project. Clients show their own label for
// the manuscript kind, in the reader's language.
const manuscriptLayerName = "Manuscript"

// projectCreateOwnIDs counts the IDs a new project needs besides its scenes': its own and its
// manuscript layer's.
const projectCreateOwnIDs = 2

// ProjectCreateDaoProject inserts the project.
type ProjectCreateDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectInsertRequest) (*dao.Project, error)
}

// ProjectCreateDaoLayer inserts the project's manuscript layer.
type ProjectCreateDaoLayer interface {
	Exec(ctx context.Context, request *dao.LayerInsertRequest) (*dao.Layer, error)
}

// ProjectCreateDaoScenes inserts the scenes the project starts with.
type ProjectCreateDaoScenes interface {
	Exec(ctx context.Context, request *dao.SceneInsertRequest) ([]*dao.Scene, error)
}

// ProjectCreateScene is a scene a new project starts with, such as one a preset seeds.
type ProjectCreateScene struct {
	Title string `validate:"required,notblank,max=200"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color       string `validate:"required,len=7,hexcolor"`
	Description string `validate:"max=20000"`
	Prompt      string `validate:"max=20000"`
}

// ProjectCreateRequest carries a new project and the scenes it starts with.
type ProjectCreateRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	Title   string    `validate:"required,notblank,max=200"`
	Premise string    `validate:"max=20000"`
	// Language is the language the project is written in.
	Language string `validate:"required,oneof=en fr"`
	// Scenes are the scenes the project starts with, in order.
	Scenes []ProjectCreateScene `validate:"dive"`
}

// ProjectCreate creates a project with its manuscript layer and its first scenes, in one
// transaction.
type ProjectCreate struct {
	daoProject ProjectCreateDaoProject
	daoLayer   ProjectCreateDaoLayer
	daoScenes  ProjectCreateDaoScenes
	transactor transaction.Transactor
	config     config.Composer
}

func NewProjectCreate(
	daoProject ProjectCreateDaoProject,
	daoLayer ProjectCreateDaoLayer,
	daoScenes ProjectCreateDaoScenes,
	transactor transaction.Transactor,
	config config.Composer,
) *ProjectCreate {
	return &ProjectCreate{
		daoProject: daoProject,
		daoLayer:   daoLayer,
		daoScenes:  daoScenes,
		transactor: transactor,
		config:     config,
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

	if len(request.Scenes) > service.config.MaxSeedScenes {
		return nil, otel.ReportError(span, fmt.Errorf(
			"%w: %d scenes, at most %d", ErrInvalidRequest, len(request.Scenes), service.config.MaxSeedScenes,
		))
	}

	// UUIDv7 sorts by creation time, which lets ProjectList order projects by ID. The project and its manuscript
	// layer take the first two IDs.
	ids, err := newIDs(len(request.Scenes) + projectCreateOwnIDs)
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

		layer, err := service.daoLayer.Exec(ctx, &dao.LayerInsertRequest{
			ID:        ids[1],
			ProjectID: project.ID,
			Position:  0,
			Kind:      LayerKindManuscript,
			Name:      manuscriptLayerName,
			Now:       now,
		})
		if err != nil {
			return fmt.Errorf("insert manuscript layer: %w", err)
		}

		composer.Project = newProject(project)
		composer.Layers = []*Layer{newLayer(layer, 0)}

		if len(request.Scenes) == 0 {
			return nil
		}

		scenes, err := service.daoScenes.Exec(ctx, &dao.SceneInsertRequest{
			ProjectID: project.ID,
			Position:  0,
			Scenes:    lo.Map(request.Scenes, seedScene(ids[2:])),
			Now:       now,
		})
		if err != nil {
			return fmt.Errorf("insert scenes: %w", err)
		}

		slices.SortFunc(scenes, func(a, b *dao.Scene) int { return cmp.Compare(a.Position, b.Position) })
		composer.Scenes = lo.Map(scenes, newScene)

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
