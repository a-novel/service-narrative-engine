package core

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	// ErrProjectNotFound is returned when no project has the ID, or another user owns it.
	ErrProjectNotFound = errors.New("project not found")
	// ErrSceneNotFound is returned when no scene has the ID, or another user owns its project.
	ErrSceneNotFound = errors.New("scene not found")
	// ErrLayerNotFound is returned when no layer has the ID, or another user owns its project.
	ErrLayerNotFound = errors.New("layer not found")
	// ErrCellNotFound is returned when the scene and the layer of a cell belong to different
	// projects.
	ErrCellNotFound = errors.New("cell not found")
	// ErrProjectFull is returned when a project already holds as many scenes or layers as it may.
	ErrProjectFull = errors.New("project full")
	// ErrManuscriptLayer is returned by a write the manuscript layer refuses: it is unique and
	// permanent.
	ErrManuscriptLayer = errors.New("manuscript layer is permanent")
)

// checkOwner returns notFound unless ownerID owns the project, so another user's data reads as
// missing.
func checkOwner(project *dao.Project, ownerID uuid.UUID, notFound error) error {
	if project.OwnerID != ownerID {
		return fmt.Errorf("%w: project %s belongs to another user", notFound, project.ID)
	}

	return nil
}

// moveTarget returns the position an entity at from takes when it moves right after the one at
// anchor, among the positions that remain once it leaves its own.
func moveTarget(from, anchor int) int {
	if anchor < from {
		return anchor + 1
	}

	return anchor
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
