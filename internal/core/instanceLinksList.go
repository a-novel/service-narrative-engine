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

// InstanceLinksListDao reads a page of the links the instances of a project store.
type InstanceLinksListDao interface {
	Exec(ctx context.Context, request *dao.InstanceLinksListRequest) ([]*dao.InstanceLinksListRow, error)
}

// InstanceLinksListRequest selects a page of the links of the instances of a project of the caller.
type InstanceLinksListRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Limit caps the page at 500 instances. It defaults to [InstanceListLimitDefault] when zero.
	Limit int `validate:"min=0,max=500"`
	// Offset is how many instances come before the page, oldest first.
	Offset int `validate:"min=0"`
}

// InstanceLinksList reads the links the instances of a project of the caller store, page by page, oldest
// instance first. An instance that never wrote its links comes with an empty list and a nil UpdatedAt. The
// pages follow those of [InstanceList], so one page of the default size holds the links of a whole project
// within the default cap on its instances. Each page carries the total.
type InstanceLinksList struct {
	daoOwner ProjectOwnerDao
	dao      InstanceLinksListDao
}

func NewInstanceLinksList(daoOwner ProjectOwnerDao, dao InstanceLinksListDao) *InstanceLinksList {
	return &InstanceLinksList{daoOwner: daoOwner, dao: dao}
}

func (service *InstanceLinksList) Exec(
	ctx context.Context, request *InstanceLinksListRequest,
) (*InstanceLinksPage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.InstanceLinksList")
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

	rows, err := service.dao.Exec(ctx, &dao.InstanceLinksListRequest{
		ProjectID: request.ProjectID,
		Limit:     request.Limit,
		Offset:    request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list links: %w", err))
	}

	page := &InstanceLinksPage{
		Instances: lo.Map(rows, func(row *dao.InstanceLinksListRow, _ int) *InstanceLinks {
			return &InstanceLinks{InstanceID: row.InstanceID, Links: row.Links, UpdatedAt: row.UpdatedAt}
		}),
	}

	// Every row carries the total, and a page past the end carries none.
	if len(rows) > 0 {
		page.Total = rows[0].Total
	}

	return page, nil
}
