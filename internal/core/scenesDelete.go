package core

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ScenesDeleteDaoLock locks the project its owner names, so changes to its order run one at a time.
type ScenesDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// ScenesDeleteDao removes the scenes, and moves the scenes after them in their beats up.
type ScenesDeleteDao interface {
	Exec(ctx context.Context, request *dao.ScenesDeleteRequest) ([]*dao.Scene, error)
}

// ScenesDeleteRequest names the scenes to delete.
type ScenesDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// IDs lists the scenes to delete, each once, at most 100.
	IDs []uuid.UUID `validate:"required,min=1,max=100,unique,dive,required"`
}

// ScenesDelete deletes scenes with their cells, and moves the scenes after them in their beats up to close
// the gaps. It deletes all of them or none.
type ScenesDelete struct {
	daoLock    ScenesDeleteDaoLock
	dao        ScenesDeleteDao
	transactor transaction.Transactor
}

func NewScenesDelete(
	daoLock ScenesDeleteDaoLock,
	dao ScenesDeleteDao,
	transactor transaction.Transactor,
) *ScenesDelete {
	return &ScenesDelete{daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *ScenesDelete) Exec(ctx context.Context, request *ScenesDeleteRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ScenesDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("scenes.count", len(request.IDs)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	var scenes []*Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		deleted, err := service.dao.Exec(ctx, &dao.ScenesDeleteRequest{ProjectID: request.ProjectID, IDs: request.IDs})
		if err != nil {
			return fmt.Errorf("delete scenes: %w", err)
		}

		// A miss rolls the whole delete back.
		if len(deleted) != len(request.IDs) {
			return fmt.Errorf("%w: %d of %d scenes", ErrSceneNotFound, len(request.IDs)-len(deleted), len(request.IDs))
		}

		slices.SortFunc(deleted, func(a, b *dao.Scene) int { return cmp.Compare(a.Position, b.Position) })
		scenes = lo.Map(deleted, func(entity *dao.Scene, _ int) *Scene { return newScene(entity) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scenes, nil
}
