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

// ProjectUpdateRequest carries a project's new fields.
type ProjectUpdateRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
	Title   string    `validate:"required,notblank,max=200"`
	Premise string    `validate:"max=20000"`
	// Language is the language the project is written in.
	Language string `validate:"required,oneof=en fr"`
}

// ProjectUpdate replaces a project's own fields.
type ProjectUpdate struct {
	dao ProjectUpdateDao
}

func NewProjectUpdate(dao ProjectUpdateDao) *ProjectUpdate {
	return &ProjectUpdate{dao: dao}
}

func (service *ProjectUpdate) Exec(ctx context.Context, request *ProjectUpdateRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.ProjectUpdateRequest{
		ID:       request.ID,
		OwnerID:  request.OwnerID,
		Title:    request.Title,
		Premise:  request.Premise,
		Language: request.Language,
		Now:      time.Now(),
	})
	if errors.Is(err, dao.ErrProjectUpdateNotFound) {
		err = errors.Join(err, ErrProjectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update project: %w", err))
	}

	return newProject(entity), nil
}
