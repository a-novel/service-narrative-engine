package core

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// snapshotPageSize is how many rows a [ProjectSnapshotRead] reads per query. It holds a project's beats,
// scenes and layers in one page each, and the summaries of its cells in a few.
const snapshotPageSize = 2000

// ProjectSnapshotReadDaoOwner checks that the caller owns the project.
type ProjectSnapshotReadDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// ProjectSnapshotReadDaoBytes reads how many bytes the project stores.
type ProjectSnapshotReadDaoBytes interface {
	Exec(ctx context.Context, request *dao.ProjectBytesRequest) (*dao.ProjectBytes, error)
}

// ProjectSnapshotReadDaoProject reads the project.
type ProjectSnapshotReadDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// ProjectSnapshotReadDaoBeats reads a page of the project's beats.
type ProjectSnapshotReadDaoBeats interface {
	Exec(ctx context.Context, request *dao.BeatListRequest) ([]*dao.BeatListRow, error)
}

// ProjectSnapshotReadDaoScenes reads a page of the project's scenes.
type ProjectSnapshotReadDaoScenes interface {
	Exec(ctx context.Context, request *dao.SceneListRequest) ([]*dao.SceneListRow, error)
}

// ProjectSnapshotReadDaoLayers reads a page of the project's layers.
type ProjectSnapshotReadDaoLayers interface {
	Exec(ctx context.Context, request *dao.LayerListRequest) ([]*dao.LayerListRow, error)
}

// ProjectSnapshotReadDaoCells reads a page of the summaries of the project's cells.
type ProjectSnapshotReadDaoCells interface {
	Exec(ctx context.Context, request *dao.CellSummaryListRequest) ([]*dao.CellSummaryListRow, error)
}

// ProjectSnapshotReadRequest names the project to read, and its caller.
type ProjectSnapshotReadRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
}

// ProjectSnapshotRead reads what a generation plans from first: the beats, scenes and layers of a project of
// the caller, and a summary of each cell, without its content or the versions of its recap. The reads run
// outside a transaction, so a write between two pages may show. A project past [ProjectBytesMax] stored
// bytes answers [ErrContextTooLarge] before any page is read.
type ProjectSnapshotRead struct {
	daoOwner   ProjectSnapshotReadDaoOwner
	daoBytes   ProjectSnapshotReadDaoBytes
	daoProject ProjectSnapshotReadDaoProject
	daoBeats   ProjectSnapshotReadDaoBeats
	daoScenes  ProjectSnapshotReadDaoScenes
	daoLayers  ProjectSnapshotReadDaoLayers
	daoCells   ProjectSnapshotReadDaoCells
}

func NewProjectSnapshotRead(
	daoOwner ProjectSnapshotReadDaoOwner,
	daoBytes ProjectSnapshotReadDaoBytes,
	daoProject ProjectSnapshotReadDaoProject,
	daoBeats ProjectSnapshotReadDaoBeats,
	daoScenes ProjectSnapshotReadDaoScenes,
	daoLayers ProjectSnapshotReadDaoLayers,
	daoCells ProjectSnapshotReadDaoCells,
) *ProjectSnapshotRead {
	return &ProjectSnapshotRead{
		daoOwner: daoOwner, daoBytes: daoBytes, daoProject: daoProject, daoBeats: daoBeats, daoScenes: daoScenes,
		daoLayers: daoLayers, daoCells: daoCells,
	}
}

func (service *ProjectSnapshotRead) Exec(
	ctx context.Context, request *ProjectSnapshotReadRequest,
) (*ProjectSnapshot, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectSnapshotRead")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	// A project past its budget would take a context past every call's limit.
	err = keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ProjectID})
	if errors.Is(err, ErrProjectBudget) {
		err = errors.Join(err, ErrContextTooLarge)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("keep project budget: %w", err))
	}

	project, err := service.daoProject.Exec(ctx, &dao.ProjectSelectRequest{ID: request.ProjectID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select project: %w", err))
	}

	snapshot := &ProjectSnapshot{Project: newProject(project)}

	snapshot.Beats, err = readPages(func(offset int) ([]*dao.BeatListRow, error) {
		return service.daoBeats.Exec(ctx, &dao.BeatListRequest{
			ProjectID: request.ProjectID, Limit: snapshotPageSize, Offset: offset, MaxBytes: ListPageBytesMax,
		})
	}, func(row *dao.BeatListRow) (*Beat, int) { return newBeat(&row.Beat), row.Total })
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list beats: %w", err))
	}

	snapshot.Scenes, err = readPages(func(offset int) ([]*dao.SceneListRow, error) {
		return service.daoScenes.Exec(ctx, &dao.SceneListRequest{
			ProjectID: request.ProjectID, Limit: snapshotPageSize, Offset: offset, MaxBytes: ListPageBytesMax,
		})
	}, func(row *dao.SceneListRow) (*Scene, int) { return newScene(&row.Scene), row.Total })
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list scenes: %w", err))
	}

	snapshot.Layers, err = readPages(func(offset int) ([]*dao.LayerListRow, error) {
		return service.daoLayers.Exec(ctx, &dao.LayerListRequest{
			ProjectID: request.ProjectID, Limit: snapshotPageSize, Offset: offset, MaxBytes: ListPageBytesMax,
		})
	}, func(row *dao.LayerListRow) (*Layer, int) { return newLayer(&row.Layer), row.Total })
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list layers: %w", err))
	}

	snapshot.Cells, err = readPages(func(offset int) ([]*dao.CellSummaryListRow, error) {
		return service.daoCells.Exec(ctx, &dao.CellSummaryListRequest{
			ProjectID: request.ProjectID, Limit: snapshotPageSize, Offset: offset,
		})
	}, func(row *dao.CellSummaryListRow) (*CellSummary, int) { return newCellSummary(row), row.Total })
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list cells: %w", err))
	}

	snapshot.BeatRecaps = lo.FilterMap(snapshot.Beats, func(beat *Beat, _ int) (*BeatRecap, bool) {
		return &BeatRecap{
			Recap: lo.FromPtr(beat.Recap), BeatID: beat.ID, BeatUpdatedAt: beat.RecapBeatUpdatedAt,
			Sources: beat.RecapSources,
		}, beat.Recap != nil
	})

	return snapshot, nil
}

// readPages reads a paged list whole, from offset 0, until it holds the total its rows carry. Read reads
// the page at an offset, and convert turns each row into its item and the list's total.
func readPages[Row, Item any](
	read func(offset int) ([]Row, error), convert func(row Row) (*Item, int),
) ([]*Item, error) {
	var items []*Item

	for {
		rows, err := read(len(items))
		if err != nil {
			return nil, err
		}

		var total int

		items = slices.Grow(items, len(rows))

		for _, row := range rows {
			var item *Item

			item, total = convert(row)
			items = append(items, item)
		}

		// A page past the end comes back empty, and a page that reaches the total ends the list. A page may
		// stop short of the limit, at its byte bound.
		if len(rows) == 0 || len(items) >= total {
			return items, nil
		}
	}
}
