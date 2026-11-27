package handlers_test

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

var (
	ownerID    = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	projectID  = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	instanceID = uuid.MustParse("00000000-0000-0000-0000-000000000031")

	createdAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt = time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
	editedAt  = time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)
	// preciseAt keeps the microseconds the database stores, which the JSON carries.
	preciseAt = time.Date(2021, 1, 3, 8, 15, 30, 123456000, time.UTC)
	// logoChecksum is the SHA-256 of the logo's bytes, in lowercase hex.
	logoChecksum = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

	// coreProject holds a logo.
	coreProject = &core.Project{
		ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Premise: "Marie attend.", Language: "fr",
		Tags: []string{"noir", "huis clos"}, Genres: []string{"thriller"},
		CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt, LogoChecksum: &logoChecksum,
	}
	jsonProject = map[string]any{
		"id": projectID.String(), "title": "Les Heures claires", "premise": "Marie attend.", "language": "fr",
		"tags": []any{"noir", "huis clos"}, "genres": []any{"thriller"},
		"createdAt": "2021-01-01T00:00:00Z", "updatedAt": "2021-01-02T00:00:00Z", "editedAt": "2021-01-03T00:00:00Z",
		"logoChecksum": logoChecksum,
	}
	// jsonProjectFields is the JSON of the fields of coreProject, which carry no logo.
	jsonProjectFields = map[string]any{
		"id": projectID.String(), "title": "Les Heures claires", "premise": "Marie attend.", "language": "fr",
		"tags": []any{"noir", "huis clos"}, "genres": []any{"thriller"},
		"createdAt": "2021-01-01T00:00:00Z", "updatedAt": "2021-01-02T00:00:00Z", "editedAt": "2021-01-03T00:00:00Z",
	}

	// coreDescriptor is an instance without its documents, last written at preciseAt.
	coreDescriptor = &core.InstanceDescriptor{
		ID: instanceID, ProjectID: projectID, Plugin: "scenes-composer", Version: "1.2.0", Name: "Composer",
		CreatedAt: createdAt, UpdatedAt: preciseAt,
	}
	// jsonDescriptor is the JSON of coreDescriptor, whose updatedAt keeps its microseconds.
	jsonDescriptor = map[string]any{
		"id": instanceID.String(), "projectID": projectID.String(), "plugin": "scenes-composer", "version": "1.2.0",
		"name": "Composer", "createdAt": "2021-01-01T00:00:00Z", "updatedAt": "2021-01-03T08:15:30.123456Z",
	}

	// coreProjectWithDefault is coreProject once it opens on the instance of the fixtures.
	coreProjectWithDefault = &core.Project{
		ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Premise: "Marie attend.", Language: "fr",
		Tags: []string{"noir", "huis clos"}, Genres: []string{"thriller"},
		CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt,
		DefaultInstanceID: &instanceID, LogoChecksum: &logoChecksum,
	}
	jsonProjectWithDefault = map[string]any{
		"id": projectID.String(), "title": "Les Heures claires", "premise": "Marie attend.", "language": "fr",
		"tags": []any{"noir", "huis clos"}, "genres": []any{"thriller"},
		"createdAt": "2021-01-01T00:00:00Z", "updatedAt": "2021-01-02T00:00:00Z", "editedAt": "2021-01-03T00:00:00Z",
		"defaultInstanceID": instanceID.String(), "logoChecksum": logoChecksum,
	}
	// jsonProjectFieldsWithDefault is the JSON of the fields of coreProjectWithDefault, which carry no logo.
	jsonProjectFieldsWithDefault = map[string]any{
		"id": projectID.String(), "title": "Les Heures claires", "premise": "Marie attend.", "language": "fr",
		"tags": []any{"noir", "huis clos"}, "genres": []any{"thriller"},
		"createdAt": "2021-01-01T00:00:00Z", "updatedAt": "2021-01-02T00:00:00Z", "editedAt": "2021-01-03T00:00:00Z",
		"defaultInstanceID": instanceID.String(),
	}
)
