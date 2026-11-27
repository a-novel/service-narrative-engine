package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/v2/internal/config"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// InstanceContentUpdateDao writes the version and the content of the instance, only while the instance still
// holds the UpdatedAt the request names.
type InstanceContentUpdateDao interface {
	Exec(ctx context.Context, request *dao.InstanceContentUpdateRequest) (*dao.InstanceDescriptor, error)
}

// InstanceContentUpdateRequest carries the new version and content of an instance.
type InstanceContentUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// Version is the exact version of the plugin the content follows, as [parsePluginVersion] reads one. A
	// client that migrates the content writes the version it migrated to.
	Version string `validate:"required,pluginversion"`
	// Content is the JSON document the plugin edits, within the configured caps.
	Content json.RawMessage `validate:"required"`

	// UpdatedAt is the UpdatedAt the instance carried when the caller read it, to the microsecond.
	UpdatedAt time.Time `validate:"required"`
}

// InstanceContentUpdate replaces the version and the content of an instance in one write, and returns the
// instance without its content. The write applies only while the instance still carries the UpdatedAt the
// request names: it merges nothing, and returns [ErrInstanceStale] once another write changed the instance. It
// writes nothing when the project would store more than [ProjectBytesMax] bytes.
//
// The service checks the size and the shape of the content, never what it means. Its span carries the bytes of
// the content, so the distribution of the write sizes shows when writing a whole document stops scaling.
type InstanceContentUpdate struct {
	daoLock    ProjectLockDao
	dao        InstanceContentUpdateDao
	daoBytes   ProjectBytesDao
	transactor transaction.Transactor
	caps       config.Caps
}

func NewInstanceContentUpdate(
	daoLock ProjectLockDao,
	dao InstanceContentUpdateDao,
	daoBytes ProjectBytesDao,
	transactor transaction.Transactor,
	caps config.Caps,
) *InstanceContentUpdate {
	return &InstanceContentUpdate{daoLock: daoLock, dao: dao, daoBytes: daoBytes, transactor: transactor, caps: caps}
}

func (service *InstanceContentUpdate) Exec(
	ctx context.Context, request *InstanceContentUpdateRequest,
) (*InstanceDescriptor, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.InstanceContentUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("instance.id", request.ID.String()),
		attribute.Int("instance.content_bytes", len(request.Content)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	err = checkDocument("content", request.Content, service.caps.ContentBytes, service.caps.DocumentDepth)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	var entity *dao.InstanceDescriptor

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		entity, err = service.dao.Exec(ctx, &dao.InstanceContentUpdateRequest{
			ProjectID: request.ProjectID,
			ID:        request.ID,
			Version:   request.Version,
			Content:   request.Content,
			UpdatedAt: request.UpdatedAt,
			Now:       time.Now(),
		})
		if err != nil {
			return fmt.Errorf("update instance content: %w", err)
		}

		err = keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("keep project budget: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return newInstanceDescriptor(entity), nil
}
