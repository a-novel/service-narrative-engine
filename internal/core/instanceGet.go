package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// InstanceGetDao reads the instance.
type InstanceGetDao interface {
	Exec(ctx context.Context, request *dao.InstanceSelectRequest) (*dao.Instance, error)
}

// InstanceGetRequest names an instance of a project of the caller.
type InstanceGetRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`
}

// InstanceGet reads an instance of a project of the caller, with its content.
type InstanceGet struct {
	daoOwner ProjectOwnerDao
	dao      InstanceGetDao
}

func NewInstanceGet(daoOwner ProjectOwnerDao, dao InstanceGetDao) *InstanceGet {
	return &InstanceGet{daoOwner: daoOwner, dao: dao}
}

func (service *InstanceGet) Exec(ctx context.Context, request *InstanceGetRequest) (*Instance, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.InstanceGet")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("instance.id", request.ID.String()),
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

	entity, err := service.dao.Exec(ctx, &dao.InstanceSelectRequest{ProjectID: request.ProjectID, ID: request.ID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select instance: %w", err))
	}

	return &Instance{
		InstanceDescriptor: *newInstanceDescriptor(&entity.InstanceDescriptor),
		Content:            entity.Content,
	}, nil
}
