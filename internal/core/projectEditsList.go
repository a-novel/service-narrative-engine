package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectEditsListDao reads when each project of the caller last changed.
type ProjectEditsListDao interface {
	Exec(ctx context.Context, request *dao.ProjectEditsListRequest) ([]*dao.ProjectEditsListRow, error)
}

// ProjectEditsListRequest names the caller whose edits to read.
type ProjectEditsListRequest struct {
	OwnerID uuid.UUID `validate:"required"`
}

// ProjectEdits holds when each project of the caller last changed.
type ProjectEdits struct {
	// Edits maps the ID of each project of the caller to its EditedAt.
	Edits map[uuid.UUID]time.Time
}

// ProjectEditsList reads the EditedAt of every project of the caller. A user owns at most [ProjectsPerOwnerMax]
// projects, so one read holds them all.
type ProjectEditsList struct {
	dao ProjectEditsListDao
}

func NewProjectEditsList(dao ProjectEditsListDao) *ProjectEditsList {
	return &ProjectEditsList{dao: dao}
}

func (service *ProjectEditsList) Exec(ctx context.Context, request *ProjectEditsListRequest) (*ProjectEdits, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectEditsList")
	defer span.End()

	span.SetAttributes(attribute.String("project.owner_id", request.OwnerID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	rows, err := service.dao.Exec(ctx, &dao.ProjectEditsListRequest{
		OwnerID: request.OwnerID,
		Limit:   ProjectsPerOwnerMax,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list project edits: %w", err))
	}

	return &ProjectEdits{
		Edits: lo.SliceToMap(rows, func(row *dao.ProjectEditsListRow) (uuid.UUID, time.Time) {
			return row.ID, row.EditedAt
		}),
	}, nil
}
