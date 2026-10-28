package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	// ErrProjectNotFound is returned when no project has the ID, or another user owns it.
	ErrProjectNotFound = errors.New("project not found")
	// ErrSceneNotFound is returned when the project holds no scene with the ID.
	ErrSceneNotFound = errors.New("scene not found")
	// ErrLayerNotFound is returned when the project holds no layer with the ID.
	ErrLayerNotFound = errors.New("layer not found")
	// ErrProjectFull is returned when a project already holds as many scenes or layers as it may.
	ErrProjectFull = errors.New("project full")
	// ErrRequiredLayer is returned by a delete that would leave a project without a layer of a required
	// kind, such as the manuscript.
	ErrRequiredLayer = errors.New("required layer")
)

// projectOwnerDao checks that a user owns a project. Each service names its own interface, and every
// one satisfies this.
type projectOwnerDao interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// projectOwnershipOpts names the project to check and the user who claims it.
type projectOwnershipOpts struct {
	projectID uuid.UUID
	ownerID   uuid.UUID
}

// checkProjectOwnership returns [ErrProjectNotFound] unless the user owns the project, so another
// user's project reads as missing.
func checkProjectOwnership(ctx context.Context, daoOwner projectOwnerDao, opts projectOwnershipOpts) error {
	ctx, span := otel.Tracer().Start(ctx, "core.checkProjectOwnership")
	defer span.End()

	owned, err := daoOwner.Exec(ctx, &dao.ProjectOwnedRequest{ID: opts.projectID, OwnerID: opts.ownerID})
	if err != nil {
		return otel.ReportError(span, err)
	}

	if !owned {
		return otel.ReportError(span, ErrProjectNotFound)
	}

	return nil
}

// projectLockDao locks a project its owner names until the transaction ends. Each service names its own
// interface, and every one satisfies this.
type projectLockDao interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// lockOwnedProject locks the project for the rest of the transaction, so changes to its order run one at a
// time, and returns [ErrProjectNotFound] unless the user owns it. One query does both.
func lockOwnedProject(ctx context.Context, daoLock projectLockDao, opts projectOwnershipOpts) error {
	ctx, span := otel.Tracer().Start(ctx, "core.lockOwnedProject")
	defer span.End()

	owned, err := daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: opts.projectID, OwnerID: opts.ownerID})
	if err != nil {
		return otel.ReportError(span, err)
	}

	if !owned {
		return otel.ReportError(span, ErrProjectNotFound)
	}

	return nil
}

// newIDs generates count UUIDv7 identifiers.
func newIDs(ctx context.Context, count int) ([]uuid.UUID, error) {
	_, span := otel.Tracer().Start(ctx, "core.newIDs")
	defer span.End()

	ids := make([]uuid.UUID, count)

	for i := range ids {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("generate id: %w", err))
		}

		ids[i] = id
	}

	return ids, nil
}
