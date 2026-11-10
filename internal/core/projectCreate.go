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

// ProjectCreateDaoLock makes the user's project creations run one at a time, until the transaction ends.
type ProjectCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.OwnerLockRequest) error
}

// ProjectCreateDaoCount counts the user's projects.
type ProjectCreateDaoCount interface {
	Exec(ctx context.Context, request *dao.ProjectCountRequest) (*dao.ProjectCount, error)
}

// ProjectCreateDaoProject inserts the project.
type ProjectCreateDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectInsertRequest) (*dao.Project, error)
}

// ProjectCreateDaoLayers inserts the project's required layers.
type ProjectCreateDaoLayers interface {
	Exec(ctx context.Context, request *dao.LayersInsertRequest) ([]*dao.Layer, error)
}

// ProjectCreateRequest carries a new project.
type ProjectCreateRequest struct {
	OwnerID uuid.UUID `validate:"required"`

	// Title names the narrative.
	Title string `validate:"required,notblank,max=80"`
	// Premise says what the narrative is about, in a few sentences.
	Premise string `validate:"max=20000"`
	// Language is the language the project is written in, one of [ProjectLanguages].
	Language string `validate:"required,language"`
	// Tags are the author's custom labels, in order, none when nil. A tag holds 1 to 24 runes and no
	// whitespace at either end, and appears once.
	Tags []string `validate:"max=10,unique,dive,required,max=24,projectlabel"`
	// Genres are the genres the author names, in order, none when nil. A genre holds 1 to 24 runes and no
	// whitespace at either end, and appears once.
	Genres []string `validate:"max=10,unique,dive,required,max=24,projectlabel"`
}

// ProjectCreate creates a project with its required layers, in one transaction. It creates nothing when the
// user already owns [ProjectsPerOwnerMax] projects.
type ProjectCreate struct {
	daoLock    ProjectCreateDaoLock
	daoCount   ProjectCreateDaoCount
	daoProject ProjectCreateDaoProject
	daoLayers  ProjectCreateDaoLayers
	transactor transaction.Transactor
}

func NewProjectCreate(
	daoLock ProjectCreateDaoLock,
	daoCount ProjectCreateDaoCount,
	daoProject ProjectCreateDaoProject,
	daoLayers ProjectCreateDaoLayers,
	transactor transaction.Transactor,
) *ProjectCreate {
	return &ProjectCreate{
		daoLock: daoLock, daoCount: daoCount, daoProject: daoProject, daoLayers: daoLayers, transactor: transactor,
	}
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

	kinds := slices.Sorted(maps.Keys(requiredLayers))
	layerItems := make([]dao.LayersInsertItem, len(kinds))

	for i, kind := range kinds {
		layerItems[i] = dao.LayersInsertItem{ID: ids[1+i], Kind: kind, Name: requiredLayers[kind]}
	}

	now := time.Now()

	var composer *ProjectComposer

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		// The lock holds the count until the insert commits, so two creations cannot both pass it.
		err := service.daoLock.Exec(ctx, &dao.OwnerLockRequest{OwnerID: request.OwnerID})
		if err != nil {
			return fmt.Errorf("lock owner: %w", err)
		}

		owned, err := service.daoCount.Exec(ctx, &dao.ProjectCountRequest{OwnerID: request.OwnerID})
		if err != nil {
			return fmt.Errorf("count projects: %w", err)
		}

		if owned.Count >= ProjectsPerOwnerMax {
			return fmt.Errorf("%w: %d projects", ErrProjectsFull, owned.Count)
		}

		project, err := service.daoProject.Exec(ctx, &dao.ProjectInsertRequest{
			ID:       ids[0],
			OwnerID:  request.OwnerID,
			Title:    request.Title,
			Premise:  request.Premise,
			Language: request.Language,
			Tags:     lo.CoalesceSliceOrEmpty(request.Tags),
			Genres:   lo.CoalesceSliceOrEmpty(request.Genres),
			Now:      now,
		})
		if err != nil {
			return fmt.Errorf("insert project: %w", err)
		}

		layers, err := service.daoLayers.Exec(ctx, &dao.LayersInsertRequest{
			ProjectID: project.ID,
			Position:  0,
			Layers:    layerItems,
			Now:       now,
		})
		if err != nil {
			return fmt.Errorf("insert required layers: %w", err)
		}

		slices.SortFunc(layers, func(a, b *dao.Layer) int { return cmp.Compare(a.Position, b.Position) })

		composer = &ProjectComposer{
			Project: newProject(project),
			Layers:  lo.Map(layers, func(entity *dao.Layer, _ int) *Layer { return newLayer(entity) }),
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return composer, nil
}
