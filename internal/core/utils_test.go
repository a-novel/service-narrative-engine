package core_test

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	ownerID   = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	projectID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	beatID1   = uuid.MustParse("00000000-0000-0000-0000-000000000031")
	beatID2   = uuid.MustParse("00000000-0000-0000-0000-000000000032")
	beatID3   = uuid.MustParse("00000000-0000-0000-0000-000000000033")
	sceneID1  = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	sceneID2  = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	sceneID3  = uuid.MustParse("00000000-0000-0000-0000-000000000013")
	layerID1  = uuid.MustParse("00000000-0000-0000-0000-000000000021")
	layerID2  = uuid.MustParse("00000000-0000-0000-0000-000000000022")
	layerID3  = uuid.MustParse("00000000-0000-0000-0000-000000000023")

	// beatColor is the color of every test beat.
	beatColor = "#4a90d9"

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

// freshIDs reports whether each ID is set and appears once, as the IDs a service generates.
func freshIDs(ids []uuid.UUID) bool {
	return !slices.Contains(ids, uuid.Nil) && len(lo.Uniq(ids)) == len(ids)
}

func daoBeat(id uuid.UUID, position int) *dao.Beat {
	return &dao.Beat{
		ID: id, ProjectID: projectID, Position: position, Title: "Beat", Color: beatColor, UpdatedAt: updatedAt,
	}
}

// daoScene returns a scene of the first test beat.
func daoScene(id uuid.UUID, position int) *dao.Scene {
	return &dao.Scene{
		ID: id, ProjectID: projectID, BeatID: beatID1, Position: position, Title: "Scene", UpdatedAt: updatedAt,
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

func coreBeat(id uuid.UUID, position int) *core.Beat {
	return &core.Beat{
		ID: id, ProjectID: projectID, Position: position, Title: "Beat", Color: beatColor, UpdatedAt: updatedAt,
	}
}

// coreScene returns a scene of the first test beat.
func coreScene(id uuid.UUID, position int) *core.Scene {
	return &core.Scene{
		ID: id, ProjectID: projectID, BeatID: beatID1, Position: position, Title: "Scene", UpdatedAt: updatedAt,
	}
}

func coreLayer(id uuid.UUID, position int) *core.Layer {
	return &core.Layer{
		ID: id, ProjectID: projectID, Position: position, Kind: core.LayerKindNotes, Name: "Layer",
		UpdatedAt: updatedAt,
	}
}
