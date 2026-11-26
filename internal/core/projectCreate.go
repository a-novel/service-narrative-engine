package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// ProjectCreateDaoLock makes the user's project creations run one at a time, until the transaction ends.
type ProjectCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.OwnerLockRequest) error
}

// ProjectCreateDaoCount counts the user's projects.
type ProjectCreateDaoCount interface {
	Exec(ctx context.Context, request *dao.ProjectCountRequest) (*dao.ProjectCount, error)
}

// ProjectCreateDaoProject inserts the project.
type ProjectCreateDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectInsertRequest) (*dao.Project, error)
}

// ProjectCreateRequest carries a new project.
type ProjectCreateRequest struct {
	OwnerID uuid.UUID `validate:"required"`

	// Title names the narrative.
	Title string `validate:"required,notblank,max=80"`
	// Premise says what the narrative is about, in a few sentences.
	Premise string `validate:"max=20000"`
	// Language is the language the project is written in, one of [ProjectLanguages].
	Language string `validate:"required,language"`
	// Tags are the author's custom labels, in order, none when nil. A tag holds 1 to 24 runes in lower kebab
	// case, as "années-50" is, and appears once.
	Tags []string `validate:"max=10,unique,dive,required,max=24,projecttag"`
	// Genres are the genres the author names, in order, none when nil. A genre holds 1 to 24 runes and no
	// whitespace at either end, and appears once.
	Genres []string `validate:"max=10,unique,dive,required,max=24,projectlabel"`
}

// ProjectCreate creates a project. It creates nothing when the user already owns [ProjectsPerOwnerMax]
// projects.
type ProjectCreate struct {
	daoLock    ProjectCreateDaoLock
	daoCount   ProjectCreateDaoCount
	daoProject ProjectCreateDaoProject
	transactor transaction.Transactor
}

func NewProjectCreate(
	daoLock ProjectCreateDaoLock,
	daoCount ProjectCreateDaoCount,
	daoProject ProjectCreateDaoProject,
	transactor transaction.Transactor,
) *ProjectCreate {
	return &ProjectCreate{daoLock: daoLock, daoCount: daoCount, daoProject: daoProject, transactor: transactor}
}

func (service *ProjectCreate) Exec(ctx context.Context, request *ProjectCreateRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectCreate")
	defer span.End()

	span.SetAttributes(attribute.String("project.owner_id", request.OwnerID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// uuid.NewV7 fails only when crypto/rand does, which Go documents as never happening on supported platforms.
	id := uuid.Must(uuid.NewV7())
	now := time.Now()

	var project *dao.Project

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		// The lock holds the count until the insert commits, so two creations cannot both pass it.
		err := service.daoLock.Exec(ctx, &dao.OwnerLockRequest{OwnerID: request.OwnerID})
		if err != nil {
			return fmt.Errorf("lock owner: %w", err)
		}

		owned, err := service.daoCount.Exec(ctx, &dao.ProjectCountRequest{OwnerID: request.OwnerID})
		if err != nil {
			return fmt.Errorf("count projects: %w", err)
		}

		if owned.Count >= ProjectsPerOwnerMax {
			return fmt.Errorf("%w: %d projects", ErrProjectsFull, owned.Count)
		}

		project, err = service.daoProject.Exec(ctx, &dao.ProjectInsertRequest{
			ID:       id,
			OwnerID:  request.OwnerID,
			Title:    request.Title,
			Premise:  request.Premise,
			Language: request.Language,
			Tags:     lo.CoalesceSliceOrEmpty(request.Tags),
			Genres:   lo.CoalesceSliceOrEmpty(request.Genres),
			Now:      now,
		})
		if err != nil {
			return fmt.Errorf("insert project: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return newProject(project), nil
}
