package services

import (
	"time"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models"
)

type Module struct {
	ID          string
	Namespace   string
	Version     string
	Preversion  string
	Description string
	Schema      jsonschema.Schema
	UI          models.ModuleUi
	CreatedAt   time.Time
}

func loadModule(module *dao.Module) *Module {
	return &Module{
		ID:          module.ID,
		Namespace:   module.Namespace,
		Version:     module.Version,
		Preversion:  module.Preversion,
		Description: module.Description,
		Schema:      module.Schema,
		UI:          module.UI,
		CreatedAt:   module.CreatedAt,
	}
}

type ModuleVersion struct {
	Version    string
	Preversion string
	CreatedAt  time.Time
}

func loadModuleVersion(module *dao.ModuleVersion) *ModuleVersion {
	return &ModuleVersion{
		Version:    module.Version,
		Preversion: module.Preversion,
		CreatedAt:  module.CreatedAt,
	}
}

func loadModuleVersionsMap(d *dao.ModuleVersion, _ int) *ModuleVersion {
	return loadModuleVersion(d)
}
