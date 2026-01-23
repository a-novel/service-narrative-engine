package handlers

import (
	"time"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/a-novel/service-narrative-engine/internal/models"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

type Module struct {
	ID          string            `json:"id"`
	Namespace   string            `json:"namespace"`
	Version     string            `json:"version"`
	Preversion  string            `json:"preversion,omitempty"`
	Description string            `json:"description"`
	Schema      jsonschema.Schema `json:"schema"`
	UI          models.ModuleUi   `json:"ui"`
	CreatedAt   time.Time         `json:"createdAt"`
}

func loadModule(s *services.Module) Module {
	return Module{
		ID:          s.ID,
		Namespace:   s.Namespace,
		Version:     s.Version,
		Preversion:  s.Preversion,
		Description: s.Description,
		Schema:      s.Schema,
		UI:          s.UI,
		CreatedAt:   s.CreatedAt,
	}
}

type ModuleVersion struct {
	Version    string    `json:"version"`
	Preversion string    `json:"preversion,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

func loadModuleVersion(s *services.ModuleVersion) ModuleVersion {
	return ModuleVersion{
		Version:    s.Version,
		Preversion: s.Preversion,
		CreatedAt:  s.CreatedAt,
	}
}

func loadModuleVersionsMap(item *services.ModuleVersion, _ int) ModuleVersion {
	return loadModuleVersion(item)
}
