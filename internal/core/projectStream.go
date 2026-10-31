package core

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"slices"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// projectStreamPageSize is how many rows a [ProjectStream] reads per query. It bounds the memory a
// stream holds, cells included.
const projectStreamPageSize = 100

const (
	// ProjectStreamProject is the section of a project stream that holds the project itself.
	ProjectStreamProject = "project"
	// ProjectStreamBeats is the section that holds the project's beats, in narrative order.
	ProjectStreamBeats = "beats"
	// ProjectStreamScenes is the section that holds the project's scenes, in narrative order.
	ProjectStreamScenes = "scenes"
	// ProjectStreamLayers is the section that holds the project's layers, in order.
	ProjectStreamLayers = "layers"
	// ProjectStreamCells is the section that holds the project's cells, by the narrative order of their
	// scene, then the position of their layer.
	ProjectStreamCells = "cells"
	// ProjectStreamEnd is the record that ends a project stream.
	ProjectStreamEnd = "end"
)

// ProjectStreamSections lists the sections of a project stream, in the order it sends them.
var ProjectStreamSections = []string{
	ProjectStreamProject, ProjectStreamBeats, ProjectStreamScenes, ProjectStreamLayers, ProjectStreamCells,
	ProjectStreamEnd,
}

// ProjectStreamDaoOwner checks that the caller owns the project.
type ProjectStreamDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// ProjectStreamDaoProject reads the project.
type ProjectStreamDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// ProjectStreamDaoBeats reads a page of the project's beats.
type ProjectStreamDaoBeats interface {
	Exec(ctx context.Context, request *dao.BeatListRequest) ([]*dao.BeatListRow, error)
}

// ProjectStreamDaoScenes reads a page of the project's scenes.
type ProjectStreamDaoScenes interface {
	Exec(ctx context.Context, request *dao.SceneListRequest) ([]*dao.SceneListRow, error)
}

// ProjectStreamDaoLayers reads a page of the project's layers.
type ProjectStreamDaoLayers interface {
	Exec(ctx context.Context, request *dao.LayerListRequest) ([]*dao.LayerListRow, error)
}

// ProjectStreamDaoCells reads a page of the project's cells.
type ProjectStreamDaoCells interface {
	Exec(ctx context.Context, request *dao.CellListRequest) ([]*dao.CellListRow, error)
}

// ProjectStreamPosition is where a record stands in a project stream: its section, and how many records
// of the section come up to it, itself included. The project's record counts 1, and the end record 0.
type ProjectStreamPosition struct {
	// Section is one of [ProjectStreamSections].
	Section string `validate:"oneof=project beats scenes layers cells end"`
	Count   int    `validate:"min=0"`
}

// ProjectStreamRequest names the project to stream, its caller, and where the stream resumes.
type ProjectStreamRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// After is the position of the last record the caller received. Nil streams from the project.
	After *ProjectStreamPosition `validate:"omitnil"`
}

// ProjectRecord is one record of a project stream. It holds one item of its section, and none at the
// end.
type ProjectRecord struct {
	Position ProjectStreamPosition

	Project *Project
	Beat    *Beat
	Scene   *Scene
	Layer   *Layer
	Cell    *Cell
}

// ProjectStream reads a project of the caller whole, one record at a time: the project, its beats,
// scenes, layers and cells, then an end record. It reads a page of each list at a time, so a write
// between two pages may show, as it does across the pages of a list.
//
// It checks ownership before the stream opens, and returns [ErrStreamEnded] when the caller resumes
// after the end record.
type ProjectStream struct {
	daoOwner   ProjectStreamDaoOwner
	daoProject ProjectStreamDaoProject
	daoBeats   ProjectStreamDaoBeats
	daoScenes  ProjectStreamDaoScenes
	daoLayers  ProjectStreamDaoLayers
	daoCells   ProjectStreamDaoCells
}

func NewProjectStream(
	daoOwner ProjectStreamDaoOwner,
	daoProject ProjectStreamDaoProject,
	daoBeats ProjectStreamDaoBeats,
	daoScenes ProjectStreamDaoScenes,
	daoLayers ProjectStreamDaoLayers,
	daoCells ProjectStreamDaoCells,
) *ProjectStream {
	return &ProjectStream{
		daoOwner: daoOwner, daoProject: daoProject, daoBeats: daoBeats, daoScenes: daoScenes,
		daoLayers: daoLayers, daoCells: daoCells,
	}
}

func (service *ProjectStream) Exec(
	ctx context.Context, request *ProjectStreamRequest,
) (iter.Seq2[*ProjectRecord, error], error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectStream")
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

	after := ProjectStreamPosition{}
	if request.After != nil {
		after = *request.After
	}

	if after.Section == ProjectStreamEnd {
		return nil, ErrStreamEnded
	}

	return func(yield func(*ProjectRecord, error) bool) {
		service.records(ctx, request.ProjectID, after, yield)
	}, nil
}

// records yields the records after a position, until the end record. It runs after Exec returns, so it
// takes a span of its own.
func (service *ProjectStream) records(
	ctx context.Context, projectID uuid.UUID, after ProjectStreamPosition, yield func(*ProjectRecord, error) bool,
) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectStream(records)")
	defer span.End()

	yield = reportErrors(span, yield)

	// The caller's section resumes after its position, and each section after it starts from its first
	// record.
	offset := func(section string) int {
		if section == after.Section {
			return after.Count
		}

		return 0
	}

	sections := []func() bool{
		func() bool {
			if offset(ProjectStreamProject) > 0 {
				return true
			}

			project, err := service.daoProject.Exec(ctx, &dao.ProjectSelectRequest{ID: projectID})
			if err != nil {
				yield(nil, fmt.Errorf("select project: %w", err))

				return false
			}

			return yield(&ProjectRecord{
				Position: ProjectStreamPosition{Section: ProjectStreamProject, Count: 1}, Project: newProject(project),
			}, nil)
		},
		func() bool {
			return streamPages(ProjectStreamBeats, offset(ProjectStreamBeats), yield,
				func(offset int) ([]*dao.BeatListRow, error) {
					return service.daoBeats.Exec(ctx, &dao.BeatListRequest{
						ProjectID: projectID, Limit: projectStreamPageSize, Offset: offset,
					})
				},
				func(row *dao.BeatListRow, record *ProjectRecord) { record.Beat = newBeat(&row.Beat) },
			)
		},
		func() bool {
			return streamPages(ProjectStreamScenes, offset(ProjectStreamScenes), yield,
				func(offset int) ([]*dao.SceneListRow, error) {
					return service.daoScenes.Exec(ctx, &dao.SceneListRequest{
						ProjectID: projectID, Limit: projectStreamPageSize, Offset: offset,
					})
				},
				func(row *dao.SceneListRow, record *ProjectRecord) { record.Scene = newScene(&row.Scene) },
			)
		},
		func() bool {
			return streamPages(ProjectStreamLayers, offset(ProjectStreamLayers), yield,
				func(offset int) ([]*dao.LayerListRow, error) {
					return service.daoLayers.Exec(ctx, &dao.LayerListRequest{
						ProjectID: projectID, Limit: projectStreamPageSize, Offset: offset,
					})
				},
				func(row *dao.LayerListRow, record *ProjectRecord) { record.Layer = newLayer(&row.Layer) },
			)
		},
		func() bool {
			return streamPages(ProjectStreamCells, offset(ProjectStreamCells), yield,
				func(offset int) ([]*dao.CellListRow, error) {
					return service.daoCells.Exec(ctx, &dao.CellListRequest{
						ProjectID: projectID, Limit: projectStreamPageSize, Offset: offset,
					})
				},
				func(row *dao.CellListRow, record *ProjectRecord) { record.Cell = newCell(&row.Cell) },
			)
		},
	}

	start := max(slices.Index(ProjectStreamSections, after.Section), 0)

	for _, section := range sections[start:] {
		if !section() {
			return
		}
	}

	yield(&ProjectRecord{Position: ProjectStreamPosition{Section: ProjectStreamEnd}}, nil)
}

// reportErrors reports on span each error the stream yields.
func reportErrors[Item any](span trace.Span, yield func(Item, error) bool) func(Item, error) bool {
	return func(item Item, err error) bool {
		if err != nil {
			err = otel.ReportError(span, err)
		}

		return yield(item, err)
	}
}

// streamPages yields the records of a paged list from offset on, one page at a time. Read reads the page
// at an offset, and fill sets the record's item from a row. It returns false once the caller stops or a
// read fails.
func streamPages[Row any](
	section string,
	offset int,
	yield func(*ProjectRecord, error) bool,
	read func(offset int) ([]Row, error),
	fill func(row Row, record *ProjectRecord),
) bool {
	for {
		rows, err := read(offset)
		if err != nil {
			yield(nil, fmt.Errorf("list %s: %w", section, err))

			return false
		}

		for _, row := range rows {
			offset++

			record := &ProjectRecord{Position: ProjectStreamPosition{Section: section, Count: offset}}
			fill(row, record)

			if !yield(record, nil) {
				return false
			}
		}

		if len(rows) < projectStreamPageSize {
			return true
		}
	}
}
