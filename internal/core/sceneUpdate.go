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

// SceneUpdateDao writes the scene's new fields.
type SceneUpdateDao interface {
	Exec(ctx context.Context, request *dao.SceneUpdateRequest) (*dao.Scene, error)
}

// SceneUpdateDaoSelect reads the scene when the write missed, to tell a deleted scene from a stale
// revision.
type SceneUpdateDaoSelect interface {
	Exec(ctx context.Context, request *dao.SceneSelectRequest) (*dao.Scene, error)
}

// SceneUpdateRequest carries a scene's new fields and the revision the caller read.
type SceneUpdateRequest struct {
	OwnerID  uuid.UUID `validate:"required"`
	ID       uuid.UUID `validate:"required"`
	Revision int       `validate:"min=1"`
	Title    string    `validate:"max=200"`
	Synopsis string    `validate:"max=4000"`
}

// SceneUpdate replaces a scene's title and synopsis. It answers [ErrRevisionConflict] when the scene
// changed since the caller read it.
type SceneUpdate struct {
	dao       SceneUpdateDao
	daoSelect SceneUpdateDaoSelect
}

func NewSceneUpdate(dao SceneUpdateDao, daoSelect SceneUpdateDaoSelect) *SceneUpdate {
	return &SceneUpdate{dao: dao, daoSelect: daoSelect}
}

func (service *SceneUpdate) Exec(ctx context.Context, request *SceneUpdateRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("scene.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("scene.revision", request.Revision),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.SceneUpdateRequest{
		ID:       request.ID,
		OwnerID:  request.OwnerID,
		Revision: request.Revision,
		Title:    request.Title,
		Synopsis: request.Synopsis,
		Now:      time.Now(),
	})
	if errors.Is(err, dao.ErrSceneUpdateNotFound) {
		err = service.missed(ctx, request.ID, request.OwnerID, err)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update scene: %w", err))
	}

	return newScene(entity, 0), nil
}

// missed tells why the write matched no row: the scene is gone, or its revision moved on.
func (service *SceneUpdate) missed(ctx context.Context, id, ownerID uuid.UUID, err error) error {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneUpdate(missed)")
	defer span.End()

	_, selectErr := service.daoSelect.Exec(ctx, &dao.SceneSelectRequest{ID: id, OwnerID: ownerID})
	if errors.Is(selectErr, dao.ErrSceneSelectNotFound) {
		return otel.ReportError(span, errors.Join(err, ErrSceneNotFound))
	}

	if selectErr != nil {
		return otel.ReportError(span, fmt.Errorf("select scene: %w", selectErr))
	}

	return otel.ReportError(span, errors.Join(err, ErrRevisionConflict))
}
