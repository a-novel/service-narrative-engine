package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/picture"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectLogoUpdateDaoOwner checks that the user owns the project.
type ProjectLogoUpdateDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// ProjectLogoUpdateDao writes the logo and the instant it changed.
type ProjectLogoUpdateDao interface {
	Exec(ctx context.Context, request *dao.ProjectLogoUpsertRequest) (*dao.Project, error)
}

// ProjectLogoUpdateRequest carries a new logo of a project.
type ProjectLogoUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Data holds the encoded image.
	Data []byte `validate:"required"`
}

// ProjectLogoUpdate replaces the logo of a project. It accepts a still WebP, a JPEG or a still PNG that decodes
// whole, of any aspect ratio, up to 128 pixels tall and 256 KiB, and refuses any other with a
// [picture.RefusedError]. A logo is cosmetic: it does not count in the project's stored bytes, and its write
// leaves the project's EditedAt as it was.
type ProjectLogoUpdate struct {
	daoOwner ProjectLogoUpdateDaoOwner
	dao      ProjectLogoUpdateDao
}

func NewProjectLogoUpdate(daoOwner ProjectLogoUpdateDaoOwner, dao ProjectLogoUpdateDao) *ProjectLogoUpdate {
	return &ProjectLogoUpdate{daoOwner: daoOwner, dao: dao}
}

func (service *ProjectLogoUpdate) Exec(ctx context.Context, request *ProjectLogoUpdateRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectLogoUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	mediaType, err := picture.Check(ctx, request.Data, projectLogoRule)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check logo: %w", err))
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	entity, err := service.dao.Exec(ctx, &dao.ProjectLogoUpsertRequest{
		ProjectID: request.ProjectID,
		Data:      request.Data,
		MediaType: mediaType,
		Now:       time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("upsert logo: %w", err))
	}

	return newProject(entity), nil
}
