package core_test

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	ownerID   = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	projectID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sceneID1  = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	sceneID2  = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	sceneID3  = uuid.MustParse("00000000-0000-0000-0000-000000000013")
	layerID1  = uuid.MustParse("00000000-0000-0000-0000-000000000021")
	layerID2  = uuid.MustParse("00000000-0000-0000-0000-000000000022")
	layerID3  = uuid.MustParse("00000000-0000-0000-0000-000000000023")

	createdAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt = time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
)

// daoMock is the answer a DAO mock gives. Go allows no generic type inside a function, so the
// tests share this one.
type daoMock[Resp any] struct {
	resp Resp
	err  error
}

// testLimits holds small limits, so a case can reach them.
var testLimits = config.Composer{MaxScenes: 3, MaxLayers: 3, MaxSeedScenes: 2, MaxCellBytes: 64}

// recent reports whether a request's instant was captured during the test.
func recent(instant time.Time) bool {
	return time.Since(instant) < time.Minute
}

func daoScene(id uuid.UUID, position int) *dao.Scene {
	return &dao.Scene{ID: id, ProjectID: projectID, Position: position, Title: "Scene", Revision: 1, UpdatedAt: updatedAt}
}

func daoLayer(id uuid.UUID, position int, kind string) *dao.Layer {
	return &dao.Layer{
		ID: id, ProjectID: projectID, Position: position, Kind: kind, Name: "Layer", Revision: 1, UpdatedAt: updatedAt,
	}
}

func notesLayer(id uuid.UUID, position int) *dao.Layer {
	return daoLayer(id, position, "notes")
}

func daoProject() *dao.Project {
	return &dao.Project{
		ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
		Revision: 1, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}
