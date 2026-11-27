package handlers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

// InstanceDescriptor is the REST representation of an instance without its content.
type InstanceDescriptor struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"projectID"`

	// Plugin is the name of the plugin, as the registry holds it.
	Plugin string `json:"plugin"`
	// Version is the exact version of the plugin the content follows, as a semantic version.
	Version string `json:"version"`
	// Name is the display name the writer gave the instance.
	Name string `json:"name"`

	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is when the version or the content last changed. Its JSON keeps the microseconds the service
	// stores, and a content write sends it back as it read it.
	UpdatedAt time.Time `json:"updatedAt"`
}

// InstancePage is the REST representation of a page of the instances of a project.
type InstancePage struct {
	Instances []InstanceDescriptor `json:"instances"`

	// Total counts every instance of the project, so a client paging through them knows when it holds them
	// all.
	Total int `json:"total"`
}

// Instance is the REST representation of an instance, with its content.
type Instance struct {
	InstanceDescriptor

	// Content is the JSON document the plugin edits.
	Content json.RawMessage `json:"content"`
}

func loadInstanceDescriptor(instance *core.InstanceDescriptor) InstanceDescriptor {
	return InstanceDescriptor{
		ID:        instance.ID,
		ProjectID: instance.ProjectID,
		Plugin:    instance.Plugin,
		Version:   instance.Version,
		Name:      instance.Name,
		CreatedAt: instance.CreatedAt,
		UpdatedAt: instance.UpdatedAt,
	}
}

func loadInstancePage(page *core.InstancePage) InstancePage {
	return InstancePage{
		Instances: lo.Map(page.Instances, func(instance *core.InstanceDescriptor, _ int) InstanceDescriptor {
			return loadInstanceDescriptor(instance)
		}),
		Total: page.Total,
	}
}

// instanceLocation returns the path of the read of an instance, which the Location of its creation names.
func instanceLocation(instance *core.InstanceDescriptor) string {
	return "/v1/instance?projectID=" + instance.ProjectID.String() + "&id=" + instance.ID.String()
}
