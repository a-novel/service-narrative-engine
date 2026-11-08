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

// ProjectPictureUpdateDaoOwner checks that the user owns the project.
type ProjectPictureUpdateDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// ProjectPictureUpdateDao writes the picture and its version.
type ProjectPictureUpdateDao interface {
	Exec(ctx context.Context, request *dao.ProjectPictureUpsertRequest) (*dao.Project, error)
}

// ProjectPictureUpdateRequest carries a new picture of a project.
type ProjectPictureUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Picture names the picture to replace: [ProjectPictureLogo] or [ProjectPictureBackground].
	Picture string `validate:"required,projectpicture"`
	// Data holds the encoded image: a logo holds 256 KiB at most, a background 512 KiB, as the schema caps
	// them.
	Data []byte `validate:"required,picturebytes"`
}

// ProjectPictureUpdate replaces a picture of a project. It accepts a still WebP or a JPEG that decodes whole
// into the picture's ratio, a square logo of 1 to 512 pixels or a 16:9 background 16 to 1280 pixels wide,
// and refuses any other with a [picture.RefusedError]. A picture is cosmetic: it does not count in the
// project's stored bytes, and its write leaves the project's EditedAt as it was.
type ProjectPictureUpdate struct {
	daoOwner ProjectPictureUpdateDaoOwner
	dao      ProjectPictureUpdateDao
}

func NewProjectPictureUpdate(daoOwner ProjectPictureUpdateDaoOwner, dao ProjectPictureUpdateDao) *ProjectPictureUpdate {
	return &ProjectPictureUpdate{daoOwner: daoOwner, dao: dao}
}

func (service *ProjectPictureUpdate) Exec(ctx context.Context, request *ProjectPictureUpdateRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectPictureUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("project.picture", request.Picture),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	mediaType, err := picture.Check(ctx, request.Data, projectPictureContracts[request.Picture].rule)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check picture: %w", err))
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	entity, err := service.dao.Exec(ctx, &dao.ProjectPictureUpsertRequest{
		ProjectID: request.ProjectID,
		Picture:   request.Picture,
		Data:      request.Data,
		MediaType: mediaType,
		Now:       time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("upsert picture: %w", err))
	}

	return newProject(entity), nil
}
