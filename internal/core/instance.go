package core

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// InstanceNameMax caps the display name of an instance, in characters.
const InstanceNameMax = 80

// InstanceVersionMax caps the version an instance pins, in characters. It is the longest version node-semver
// reads (its MAX_LENGTH), so no JavaScript tooling parses a longer one.
const InstanceVersionMax = 256

// InstanceDescriptor is what a project knows of an instance without reading its content.
type InstanceDescriptor struct {
	ID        uuid.UUID
	ProjectID uuid.UUID

	// Plugin is the name of the plugin, as the registry holds it.
	Plugin string
	// Version is the exact version of the plugin the content follows, as a semantic version.
	Version string
	// Name is the display name the writer gave the instance, empty when they gave none.
	Name string

	CreatedAt time.Time
	// UpdatedAt is when the version or the content last changed, to the microsecond. A content write names it,
	// and a rename leaves it as it was.
	UpdatedAt time.Time
}

// InstancePage is a page of the instances of a project.
type InstancePage struct {
	Instances []*InstanceDescriptor

	// Total counts every instance of the project, so a client paging through them knows when it holds them
	// all. A page past the end reports 0.
	Total int
}

// Instance is one use of a plugin in a project, with its content. The service stores the content and never
// reads what it means.
type Instance struct {
	InstanceDescriptor

	// Content is the JSON document the plugin edits.
	Content json.RawMessage
}

func newInstanceDescriptor(entity *dao.InstanceDescriptor) *InstanceDescriptor {
	return &InstanceDescriptor{
		ID:        entity.ID,
		ProjectID: entity.ProjectID,
		Plugin:    entity.Plugin,
		Version:   entity.Version,
		Name:      entity.Name,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}
