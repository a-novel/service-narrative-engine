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
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectPictureUpdateDaoLock locks the project its owner names, so the picture write stamps the project's
// edit.
type ProjectPictureUpdateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
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
// and refuses any other with a [picture.RefusedError]. The picture does not count in the project's stored
// bytes.
type ProjectPictureUpdate struct {
	daoLock    ProjectPictureUpdateDaoLock
	dao        ProjectPictureUpdateDao
	transactor transaction.Transactor
}

func NewProjectPictureUpdate(
	daoLock ProjectPictureUpdateDaoLock, dao ProjectPictureUpdateDao, transactor transaction.Transactor,
) *ProjectPictureUpdate {
	return &ProjectPictureUpdate{daoLock: daoLock, dao: dao, transactor: transactor}
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

	// The picture decodes before the transaction, so the project's lock never waits on a decoder.
	mediaType, err := picture.Check(ctx, request.Data, projectPictureContracts[request.Picture].rule)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check picture: %w", err))
	}

	var project *Project

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		entity, err := service.dao.Exec(ctx, &dao.ProjectPictureUpsertRequest{
			ProjectID: request.ProjectID,
			Picture:   request.Picture,
			Data:      request.Data,
			MediaType: mediaType,
			Now:       time.Now(),
		})
		if err != nil {
			return fmt.Errorf("upsert picture: %w", err)
		}

		project = newProject(entity)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return project, nil
}
