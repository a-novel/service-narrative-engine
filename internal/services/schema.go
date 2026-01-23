package services

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

type Schema struct {
	ID               uuid.UUID
	ProjectID        uuid.UUID
	Owner            *uuid.UUID
	ModuleID         string
	ModuleNamespace  string
	ModuleVersion    string
	ModulePreversion string
	Source           string
	Data             map[string]any
	CreatedAt        time.Time
}

func loadSchema(schema *dao.Schema) *Schema {
	return &Schema{
		ID:               schema.ID,
		ProjectID:        schema.ProjectID,
		Owner:            schema.Owner,
		ModuleID:         schema.ModuleID,
		ModuleNamespace:  schema.ModuleNamespace,
		ModuleVersion:    schema.ModuleVersion,
		ModulePreversion: schema.ModulePreversion,
		Source:           schema.Source.String(),
		Data:             schema.Data,
		CreatedAt:        schema.CreatedAt,
	}
}

type SchemaVersion struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

func loadSchemaVersion(s *dao.SchemaVersion) *SchemaVersion {
	return &SchemaVersion{
		ID:        s.ID,
		CreatedAt: s.CreatedAt,
	}
}

func loadSchemaVersionsMap(item *dao.SchemaVersion, _ int) *SchemaVersion {
	return loadSchemaVersion(item)
}
