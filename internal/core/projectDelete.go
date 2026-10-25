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

// ProjectDeleteDao removes the project.
type ProjectDeleteDao interface {
	Exec(ctx context.Context, request *dao.ProjectDeleteRequest) (*dao.Project, error)
}

// ProjectDeleteDaoSelect reads the project when the delete missed, to tell a deleted project from a
// stale revision.
type ProjectDeleteDaoSelect interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// ProjectDeleteRequest names the project to delete and the revision the caller read.
type ProjectDeleteRequest struct {
	OwnerID  uuid.UUID `validate:"required"`
	ID       uuid.UUID `validate:"required"`
	Revision int       `validate:"min=1"`
}

// ProjectDelete deletes a project with its scenes, layers and cells. It answers
// [ErrRevisionConflict] when the project changed since the caller read it.
type ProjectDelete struct {
	dao       ProjectDeleteDao
	daoSelect ProjectDeleteDaoSelect
}

func NewProjectDelete(dao ProjectDeleteDao, daoSelect ProjectDeleteDaoSelect) *ProjectDelete {
	return &ProjectDelete{dao: dao, daoSelect: daoSelect}
}

func (service *ProjectDelete) Exec(ctx context.Context, request *ProjectDeleteRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("project.revision", request.Revision),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.ProjectDeleteRequest{
		ID:       request.ID,
		OwnerID:  request.OwnerID,
		Revision: request.Revision,
	})
	if errors.Is(err, dao.ErrProjectDeleteNotFound) {
		err = service.missed(ctx, request.ID, request.OwnerID, err)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("delete project: %w", err))
	}

	return newProject(entity), nil
}

// missed tells why the delete matched no row: the project is gone, or its revision moved on.
func (service *ProjectDelete) missed(ctx context.Context, id, ownerID uuid.UUID, err error) error {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectDelete(missed)")
	defer span.End()

	_, selectErr := service.daoSelect.Exec(ctx, &dao.ProjectSelectRequest{ID: id, OwnerID: ownerID})
	if errors.Is(selectErr, dao.ErrProjectSelectNotFound) {
		return otel.ReportError(span, errors.Join(err, ErrProjectNotFound))
	}

	if selectErr != nil {
		return otel.ReportError(span, fmt.Errorf("select project: %w", selectErr))
	}

	return otel.ReportError(span, errors.Join(err, ErrRevisionConflict))
}
