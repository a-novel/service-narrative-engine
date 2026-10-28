package core

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
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
	// ErrManuscriptLayer is returned by a write the manuscript layer refuses: it is unique and
	// permanent.
	ErrManuscriptLayer = errors.New("manuscript layer is permanent")
)

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
