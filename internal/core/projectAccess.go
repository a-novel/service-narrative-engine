package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	// ErrProjectNotFound is returned for both absent and cross-owner Projects.
	ErrProjectNotFound = errors.New("project not found")

	errProjectAccessMissing = errors.New("project selection returned no entity")
)

// ProjectSelectDao retrieves only stable owner-scoped Project metadata.
type ProjectSelectDao interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// ProjectAccessService resolves a Project only when the Actor owns it.
type ProjectAccessService interface {
	Exec(ctx context.Context, request *ProjectAccessRequest) (*dao.Project, error)
}

// ProjectAccessRequest identifies an Actor and Project.
type ProjectAccessRequest struct {
	Actor     Actor     `validate:"actor"`
	ProjectID uuid.UUID `validate:"required"`
}

// ProjectAccess centralizes the current Project ownership rule.
type ProjectAccess struct {
	projectDao ProjectSelectDao
}

// NewProjectAccess creates a Project access service.
func NewProjectAccess(projectDao ProjectSelectDao) *ProjectAccess {
	return &ProjectAccess{projectDao: projectDao}
}

// Exec returns the stable Project when the Actor owns it.
func (service *ProjectAccess) Exec(
	ctx context.Context,
	request *ProjectAccessRequest,
) (*dao.Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectAccess")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.Actor.UserID.String()),
	)

	project, err := service.projectDao.Exec(ctx, &dao.ProjectSelectRequest{
		ID:      request.ProjectID,
		OwnerID: request.Actor.UserID,
	})
	if errors.Is(err, dao.ErrProjectSelectNotFound) {
		err = errors.Join(err, ErrProjectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select Project: %w", err))
	}

	if project == nil {
		return nil, otel.ReportError(span, fmt.Errorf("select Project: %w", errProjectAccessMissing))
	}

	return otel.ReportSuccess(span, project), nil
}
