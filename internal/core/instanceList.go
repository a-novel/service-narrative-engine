package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// InstanceListDao reads a page of the instances of a project.
type InstanceListDao interface {
	Exec(ctx context.Context, request *dao.InstanceListRequest) ([]*dao.InstanceListRow, error)
}

// InstanceListLimitDefault is the page size of an [InstanceListRequest] whose Limit is zero.
const InstanceListLimitDefault = 100

// InstanceListRequest selects a page of the instances of a project of the caller.
type InstanceListRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Limit caps the page at 500 instances. It defaults to [InstanceListLimitDefault] when zero.
	Limit int `validate:"min=0,max=500"`
	// Offset is how many instances come before the page, oldest first.
	Offset int `validate:"min=0"`
}

// InstanceList reads the instances of a project of the caller page by page, oldest first, without their
// content. Each page carries the total, so a client knows when it holds every instance.
type InstanceList struct {
	daoOwner ProjectOwnerDao
	dao      InstanceListDao
}

func NewInstanceList(daoOwner ProjectOwnerDao, dao InstanceListDao) *InstanceList {
	return &InstanceList{daoOwner: daoOwner, dao: dao}
}

func (service *InstanceList) Exec(ctx context.Context, request *InstanceListRequest) (*InstancePage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.InstanceList")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("instances.limit", request.Limit),
		attribute.Int("instances.offset", request.Offset),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if request.Limit == 0 {
		request.Limit = InstanceListLimitDefault
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	rows, err := service.dao.Exec(ctx, &dao.InstanceListRequest{
		ProjectID: request.ProjectID,
		Limit:     request.Limit,
		Offset:    request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list instances: %w", err))
	}

	page := &InstancePage{
		Instances: lo.Map(rows, func(row *dao.InstanceListRow, _ int) *InstanceDescriptor {
			return newInstanceDescriptor(&row.InstanceDescriptor)
		}),
	}

	// Every row carries the total, and a page past the end carries none.
	if len(rows) > 0 {
		page.Total = rows[0].Total
	}

	return page, nil
}
