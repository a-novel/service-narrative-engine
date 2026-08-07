package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectGetIdeaDao retrieves the current Idea after Project authorization.
type ProjectGetIdeaDao interface {
	Exec(ctx context.Context, request *dao.IdeaSelectRequest) (*dao.Idea, error)
}

// ProjectGetStepValueDao retrieves one current value per opaque Step key.
type ProjectGetStepValueDao interface {
	Exec(
		ctx context.Context,
		request *dao.StepValueCurrentListRequest,
	) ([]*dao.StepValue, error)
}

// ProjectGetManuscriptDao retrieves the optional current Manuscript.
type ProjectGetManuscriptDao interface {
	Exec(ctx context.Context, request *dao.ManuscriptSelectRequest) (*dao.Manuscript, error)
}

// ProjectGetRequest identifies an owned Project snapshot.
type ProjectGetRequest struct {
	Actor     Actor     `validate:"actor"`
	ProjectID uuid.UUID `validate:"required"`
}

// ProjectSnapshot is the latest saved content for one stable Project.
type ProjectSnapshot struct {
	ID         uuid.UUID
	CreatedAt  time.Time
	Idea       *Idea
	StepValues []*StepValue
	Manuscript *Manuscript
}

// ProjectGet resolves a Project's current content after checking ownership.
type ProjectGet struct {
	projectAccess ProjectAccessService
	ideaDao       ProjectGetIdeaDao
	stepValueDao  ProjectGetStepValueDao
	manuscriptDao ProjectGetManuscriptDao
}

// NewProjectGet creates a Project snapshot service.
func NewProjectGet(
	projectAccess ProjectAccessService,
	ideaDao ProjectGetIdeaDao,
	stepValueDao ProjectGetStepValueDao,
	manuscriptDao ProjectGetManuscriptDao,
) *ProjectGet {
	return &ProjectGet{
		projectAccess: projectAccess,
		ideaDao:       ideaDao,
		stepValueDao:  stepValueDao,
		manuscriptDao: manuscriptDao,
	}
}

// Exec returns the latest Idea, each latest Step Value, and the latest Manuscript.
func (service *ProjectGet) Exec(
	ctx context.Context,
	request *ProjectGetRequest,
) (*ProjectSnapshot, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectGet")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.Actor.UserID.String()),
	)

	project, err := service.projectAccess.Exec(ctx, &ProjectAccessRequest{
		Actor:     request.Actor,
		ProjectID: request.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("access Project: %w", err))
	}

	ideaEntity, err := service.ideaDao.Exec(ctx, &dao.IdeaSelectRequest{
		ProjectID: request.ProjectID,
		OwnerID:   request.Actor.UserID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select current Idea: %w", err))
	}

	stepEntities, err := service.stepValueDao.Exec(ctx, &dao.StepValueCurrentListRequest{
		ProjectID: request.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select current Step Values: %w", err))
	}

	manuscriptEntity, err := service.manuscriptDao.Exec(ctx, &dao.ManuscriptSelectRequest{
		ProjectID: request.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select current Manuscript: %w", err))
	}

	stepValues := make([]*StepValue, 0, len(stepEntities))
	for _, entity := range stepEntities {
		stepValues = append(stepValues, stepValueFromDao(entity))
	}

	return otel.ReportSuccess(span, &ProjectSnapshot{
		ID:         project.ID,
		CreatedAt:  project.CreatedAt,
		Idea:       ideaFromDao(ideaEntity),
		StepValues: stepValues,
		Manuscript: manuscriptFromDao(manuscriptEntity),
	}), nil
}
