package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// snapshotPageSize is how many rows a [ProjectSnapshotRead] reads per query. It holds a project's beats,
// scenes and layers in one page each, and its cells in a few.
const snapshotPageSize = 2000

// ProjectSnapshotReadDaoOwner checks that the caller owns the project.
type ProjectSnapshotReadDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
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

// ProjectSnapshotReadDaoCells reads a page of the project's cells.
type ProjectSnapshotReadDaoCells interface {
	Exec(ctx context.Context, request *dao.CellListRequest) ([]*dao.CellListRow, error)
}

// ProjectSnapshotReadRequest names the project to read, and its caller.
type ProjectSnapshotReadRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
}

// ProjectSnapshotRead reads a project of the caller whole, as a generation plans from it: its beats,
// scenes, layers and cells. The reads run outside a transaction, so a write between two pages may show.
type ProjectSnapshotRead struct {
	daoOwner   ProjectSnapshotReadDaoOwner
	daoProject ProjectSnapshotReadDaoProject
	daoBeats   ProjectSnapshotReadDaoBeats
	daoScenes  ProjectSnapshotReadDaoScenes
	daoLayers  ProjectSnapshotReadDaoLayers
	daoCells   ProjectSnapshotReadDaoCells
}

func NewProjectSnapshotRead(
	daoOwner ProjectSnapshotReadDaoOwner,
	daoProject ProjectSnapshotReadDaoProject,
	daoBeats ProjectSnapshotReadDaoBeats,
	daoScenes ProjectSnapshotReadDaoScenes,
	daoLayers ProjectSnapshotReadDaoLayers,
	daoCells ProjectSnapshotReadDaoCells,
) *ProjectSnapshotRead {
	return &ProjectSnapshotRead{
		daoOwner: daoOwner, daoProject: daoProject, daoBeats: daoBeats, daoScenes: daoScenes,
		daoLayers: daoLayers, daoCells: daoCells,
	}
}

func (service *ProjectSnapshotRead) Exec(
	ctx context.Context, request *ProjectSnapshotReadRequest,
) (*ProjectSnapshot, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectSnapshotRead")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

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

	project, err := service.daoProject.Exec(ctx, &dao.ProjectSelectRequest{ID: request.ProjectID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select project: %w", err))
	}

	snapshot := &ProjectSnapshot{Project: newProject(project)}

	snapshot.Beats, err = readPages(func(offset int) ([]*dao.BeatListRow, error) {
		return service.daoBeats.Exec(ctx, &dao.BeatListRequest{
			ProjectID: request.ProjectID, Limit: snapshotPageSize, Offset: offset,
		})
	}, func(row *dao.BeatListRow) (*Beat, int) { return newBeat(&row.Beat), row.Total })
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list beats: %w", err))
	}

	snapshot.Scenes, err = readPages(func(offset int) ([]*dao.SceneListRow, error) {
		return service.daoScenes.Exec(ctx, &dao.SceneListRequest{
			ProjectID: request.ProjectID, Limit: snapshotPageSize, Offset: offset,
		})
	}, func(row *dao.SceneListRow) (*Scene, int) { return newScene(&row.Scene), row.Total })
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list scenes: %w", err))
	}

	snapshot.Layers, err = readPages(func(offset int) ([]*dao.LayerListRow, error) {
		return service.daoLayers.Exec(ctx, &dao.LayerListRequest{
			ProjectID: request.ProjectID, Limit: snapshotPageSize, Offset: offset,
		})
	}, func(row *dao.LayerListRow) (*Layer, int) { return newLayer(&row.Layer), row.Total })
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list layers: %w", err))
	}

	snapshot.Cells, err = readPages(func(offset int) ([]*dao.CellListRow, error) {
		return service.daoCells.Exec(ctx, &dao.CellListRequest{
			ProjectID: request.ProjectID, Limit: snapshotPageSize, Offset: offset,
		})
	}, func(row *dao.CellListRow) (*Cell, int) { return newCell(&row.Cell), row.Total })
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list cells: %w", err))
	}

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

		for _, row := range rows {
			var item *Item

			item, total = convert(row)
			items = append(items, item)
		}

		// A short page ends the list, and so does a page that reaches the total.
		if len(rows) < snapshotPageSize || len(items) >= total {
			return items, nil
		}
	}
}
