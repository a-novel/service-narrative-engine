package core

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
)

var (
	// ErrProjectNotFound is returned when no project with the ID belongs to the caller.
	ErrProjectNotFound = errors.New("project not found")
	// ErrSceneNotFound is returned when no scene with the ID belongs to a project of the caller.
	ErrSceneNotFound = errors.New("scene not found")
	// ErrLayerNotFound is returned when no layer with the ID belongs to a project of the caller.
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

// moveAfter returns ids with id moved right after anchor, or first when anchor is uuid.Nil.
func moveAfter(ids []uuid.UUID, id, anchor uuid.UUID) ([]uuid.UUID, error) {
	if anchor == id {
		return nil, fmt.Errorf("%w: %s cannot follow itself", ErrInvalidRequest, id)
	}

	order := slices.DeleteFunc(slices.Clone(ids), func(candidate uuid.UUID) bool { return candidate == id })

	position := 0

	if anchor != uuid.Nil {
		index := slices.Index(order, anchor)
		if index < 0 {
			return nil, fmt.Errorf("%w: unknown anchor %s", ErrInvalidRequest, anchor)
		}

		position = index + 1
	}

	return slices.Insert(order, position, id), nil
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
