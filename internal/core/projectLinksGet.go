package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// ProjectLinksGetDao reads the links of the project. It joins [dao.ErrProjectLinksNotFound] for a project that
// never wrote its links.
type ProjectLinksGetDao interface {
	Exec(ctx context.Context, request *dao.ProjectLinksSelectRequest) (*dao.ProjectLinks, error)
}

// ProjectLinksGetRequest names a project of the caller.
type ProjectLinksGetRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
}

// ProjectLinksGet reads the links of a project of the caller. A project that never wrote its links returns an
// empty list with a nil UpdatedAt.
type ProjectLinksGet struct {
	daoOwner ProjectOwnerDao
	dao      ProjectLinksGetDao
}

func NewProjectLinksGet(daoOwner ProjectOwnerDao, dao ProjectLinksGetDao) *ProjectLinksGet {
	return &ProjectLinksGet{daoOwner: daoOwner, dao: dao}
}

func (service *ProjectLinksGet) Exec(ctx context.Context, request *ProjectLinksGetRequest) (*ProjectLinks, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectLinksGet")
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

	entity, err := service.dao.Exec(ctx, &dao.ProjectLinksSelectRequest{ProjectID: request.ProjectID})
	// The ownership check proved the project exists, so it only never wrote its links.
	if errors.Is(err, dao.ErrProjectLinksNotFound) {
		return &ProjectLinks{Links: json.RawMessage("[]")}, nil
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select links: %w", err))
	}

	return &ProjectLinks{Links: entity.Links, UpdatedAt: &entity.UpdatedAt}, nil
}
