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

// ProjectFieldsDriftDao compares the projects a client holds with the caller's.
type ProjectFieldsDriftDao interface {
	Exec(ctx context.Context, request *dao.ProjectFieldsDriftRequest) ([]*dao.ProjectFieldsDriftRow, error)
}

// ProjectFieldsDriftRequest carries the projects a client holds.
type ProjectFieldsDriftRequest struct {
	OwnerID uuid.UUID `validate:"required"`

	// Projects maps the ID of each project the client holds to its UpdatedAt, as RFC 3339 text, compared to the
	// millisecond. It holds at most [ProjectsPerOwnerMax] projects, and none when the client holds none.
	Projects map[string]string `validate:"required,max=100,dive,keys,uuid,endkeys,datetime=2006-01-02T15:04:05Z07:00"`
}

// ProjectFieldsDriftResult is the drift between the projects a client holds and the caller's.
type ProjectFieldsDriftResult struct {
	// Projects are the caller's projects the request leaves out, or names with another UpdatedAt. They carry no
	// logo.
	Projects []*Project
	// Removed are the IDs of the request that name no project of the caller.
	Removed []uuid.UUID
}

// ProjectFieldsDrift compares the fields of the projects a client holds with the caller's. A project's UpdatedAt
// changes only with its fields, so the client sends that alone, and one query finds the drift.
type ProjectFieldsDrift struct {
	dao ProjectFieldsDriftDao
}

func NewProjectFieldsDrift(dao ProjectFieldsDriftDao) *ProjectFieldsDrift {
	return &ProjectFieldsDrift{dao: dao}
}

func (service *ProjectFieldsDrift) Exec(
	ctx context.Context, request *ProjectFieldsDriftRequest,
) (*ProjectFieldsDriftResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectFieldsDrift")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("projects.held", len(request.Projects)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// Each project of the owner, and each held ID that names none, makes one row at most.
	rows, err := service.dao.Exec(ctx, &dao.ProjectFieldsDriftRequest{
		OwnerID:  request.OwnerID,
		Projects: request.Projects,
		Limit:    ProjectsPerOwnerMax + len(request.Projects),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("compare project fields: %w", err))
	}

	result := &ProjectFieldsDriftResult{Projects: []*Project{}, Removed: []uuid.UUID{}}

	for _, row := range rows {
		if row.RemovedID != nil {
			result.Removed = append(result.Removed, *row.RemovedID)

			continue
		}

		result.Projects = append(result.Projects, newProject(&row.Project))
	}

	return result, nil
}
