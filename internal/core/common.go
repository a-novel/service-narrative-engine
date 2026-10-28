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
	// ErrRequiredLayer is returned by a delete of a layer every project holds, such as the manuscript.
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

// newIDs generates count UUIDv7 identifiers.
func newIDs(count int) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, count)

	for i := range ids {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("generate id: %w", err)
		}

		ids[i] = id
	}

	return ids, nil
}
