package core_test

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	ownerID      = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	otherOwnerID = uuid.MustParse("00000000-0000-0000-0000-0000000000a2")
	projectID    = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sceneID1     = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	sceneID2     = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	sceneID3     = uuid.MustParse("00000000-0000-0000-0000-000000000013")
	layerID1     = uuid.MustParse("00000000-0000-0000-0000-000000000021")
	layerID2     = uuid.MustParse("00000000-0000-0000-0000-000000000022")
	layerID3     = uuid.MustParse("00000000-0000-0000-0000-000000000023")

	// sceneColor is the color of every test scene.
	sceneColor = "#4a90d9"

	createdAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt = time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
)

// daoMock is the answer a DAO mock gives. Go allows no generic type inside a function, so the
// tests share this one.
type daoMock[Resp any] struct {
	resp Resp
	err  error
}

// recent reports whether a request's instant was captured during the test.
func recent(instant time.Time) bool {
	return time.Since(instant) < time.Minute
}

func daoScene(id uuid.UUID, position int) *dao.Scene {
	return &dao.Scene{
		ID: id, ProjectID: projectID, Position: position, Title: "Scene", Color: sceneColor, UpdatedAt: updatedAt,
	}
}

func daoLayer(id uuid.UUID, position int, kind string) *dao.Layer {
	return &dao.Layer{
		ID: id, ProjectID: projectID, Position: position, Kind: kind, Name: "Layer", UpdatedAt: updatedAt,
	}
}

func notesLayer(id uuid.UUID, position int) *dao.Layer {
	return daoLayer(id, position, "notes")
}

func daoProject() *dao.Project {
	return &dao.Project{
		ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

// foreignProject returns the test project as another user owns it.
func foreignProject() *dao.Project {
	project := daoProject()
	project.OwnerID = otherOwnerID

	return project
}

func coreScene(id uuid.UUID, position int) *core.Scene {
	return &core.Scene{
		ID: id, ProjectID: projectID, Position: position, Title: "Scene", Color: sceneColor, UpdatedAt: updatedAt,
	}
}

func coreLayer(id uuid.UUID, position int) *core.Layer {
	return &core.Layer{
		ID: id, ProjectID: projectID, Position: position, Kind: core.LayerKindNotes, Name: "Layer",
		UpdatedAt: updatedAt,
	}
}
