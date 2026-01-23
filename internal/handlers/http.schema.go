package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/lib"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

type Schema struct {
	ID        uuid.UUID      `json:"id"`
	ProjectID uuid.UUID      `json:"projectID"`
	Owner     *uuid.UUID     `json:"owner"`
	Module    string         `json:"module"`
	Source    string         `json:"source"`
	Data      map[string]any `json:"data"`
	CreatedAt time.Time      `json:"createdAt"`
}

func loadSchema(s *services.Schema) Schema {
	return Schema{
		ID:        s.ID,
		ProjectID: s.ProjectID,
		Owner:     s.Owner,
		Module: (lib.DecodedModule{
			Namespace:  s.ModuleNamespace,
			Module:     s.ModuleID,
			Version:    s.ModuleVersion,
			Preversion: s.ModulePreversion,
		}).String(),
		Source:    s.Source,
		Data:      s.Data,
		CreatedAt: s.CreatedAt,
	}
}

type SchemaVersion struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

func loadSchemaVersion(s *services.SchemaVersion) SchemaVersion {
	return SchemaVersion{
		ID:        s.ID,
		CreatedAt: s.CreatedAt,
	}
}

func loadSchemaVersionsMap(item *services.SchemaVersion, _ int) SchemaVersion {
	return loadSchemaVersion(item)
}
