package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectUpdateDao writes the project's new fields.
type ProjectUpdateDao interface {
	Exec(ctx context.Context, request *dao.ProjectUpdateRequest) (*dao.Project, error)
}

// ProjectUpdateDaoSelect reads the project when the write missed, to tell a deleted project from a
// stale revision.
type ProjectUpdateDaoSelect interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// ProjectUpdateRequest carries a project's new fields and the revision the caller read.
type ProjectUpdateRequest struct {
	OwnerID  uuid.UUID `validate:"required"`
	ID       uuid.UUID `validate:"required"`
	Revision int       `validate:"min=1"`
	Title    string    `validate:"required,notblank,max=200"`
	Premise  string    `validate:"max=4000"`
	// Language is the language the project is written in.
	Language string `validate:"required,oneof=en fr"`
}

// ProjectUpdate replaces a project's own fields. It answers [ErrRevisionConflict] when the project
// changed since the caller read it.
type ProjectUpdate struct {
	dao       ProjectUpdateDao
	daoSelect ProjectUpdateDaoSelect
}

func NewProjectUpdate(dao ProjectUpdateDao, daoSelect ProjectUpdateDaoSelect) *ProjectUpdate {
	return &ProjectUpdate{dao: dao, daoSelect: daoSelect}
}

func (service *ProjectUpdate) Exec(ctx context.Context, request *ProjectUpdateRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectUpdate")
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

	entity, err := service.dao.Exec(ctx, &dao.ProjectUpdateRequest{
		ID:       request.ID,
		OwnerID:  request.OwnerID,
		Revision: request.Revision,
		Title:    request.Title,
		Premise:  request.Premise,
		Language: request.Language,
		Now:      time.Now(),
	})
	if errors.Is(err, dao.ErrProjectUpdateNotFound) {
		err = service.missed(ctx, request.ID, request.OwnerID, err)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update project: %w", err))
	}

	return newProject(entity), nil
}

// missed tells why a write to the project matched no row: the project is gone, or its revision
// moved on.
func (service *ProjectUpdate) missed(ctx context.Context, id, ownerID uuid.UUID, err error) error {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectUpdate(missed)")
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
