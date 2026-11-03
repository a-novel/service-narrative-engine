package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectUpdateDaoLock locks the project its owner names, so the writes to its bytes run one at a time.
type ProjectUpdateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// ProjectUpdateDao writes the project's new fields.
type ProjectUpdateDao interface {
	Exec(ctx context.Context, request *dao.ProjectUpdateRequest) (*dao.Project, error)
}

// ProjectUpdateDaoBytes reads how many bytes the project stores, the write included.
type ProjectUpdateDaoBytes interface {
	Exec(ctx context.Context, request *dao.ProjectBytesRequest) (*dao.ProjectBytes, error)
}

// ProjectUpdateRequest carries a project's new fields.
type ProjectUpdateRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`

	// Title names the narrative.
	Title string `validate:"required,notblank,max=200"`
	// Premise says what the narrative is about, in a few sentences.
	Premise string `validate:"max=20000"`
	// Language is the language the project is written in, one of [ProjectLanguages].
	Language string `validate:"required,language"`
}

// ProjectUpdate replaces a project's own fields. It writes nothing when the project would store more than
// [ProjectBytesMax] bytes.
type ProjectUpdate struct {
	daoLock    ProjectUpdateDaoLock
	dao        ProjectUpdateDao
	daoBytes   ProjectUpdateDaoBytes
	transactor transaction.Transactor
}

func NewProjectUpdate(
	daoLock ProjectUpdateDaoLock,
	dao ProjectUpdateDao,
	daoBytes ProjectUpdateDaoBytes,
	transactor transaction.Transactor,
) *ProjectUpdate {
	return &ProjectUpdate{daoLock: daoLock, dao: dao, daoBytes: daoBytes, transactor: transactor}
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

	var project *Project

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		entity, err := service.dao.Exec(ctx, &dao.ProjectUpdateRequest{
			ID:       request.ID,
			Title:    request.Title,
			Premise:  request.Premise,
			Language: request.Language,
			Now:      time.Now(),
		})
		if errors.Is(err, dao.ErrProjectUpdateNotFound) {
			err = errors.Join(err, ErrProjectNotFound)
		}

		if err != nil {
			return fmt.Errorf("update project: %w", err)
		}

		err = keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ID})
		if err != nil {
			return fmt.Errorf("keep project budget: %w", err)
		}

		project = newProject(entity)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return project, nil
}
