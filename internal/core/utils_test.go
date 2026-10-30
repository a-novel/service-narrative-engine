package core_test

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

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

// genaiID names the genai generation of the generation tests.
var genaiID = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")

// suggestOutput is a valid suggest output, with one text suggestion.
const suggestOutput = `{"manuscript":[],"text":[{"target":"T1","note":"Names the scene.","text":"La lettre"}]}`

// genaiGeneration returns the genai generation of the test owner at a status. Its last attempt holds
// response when set.
func genaiGeneration(status servicegenai.GenerationStatus, response string) *servicegenai.Generation {
	generation := &servicegenai.Generation{Id: genaiID.String(), OwnerId: ownerID.String(), Status: status}
	if response != "" {
		generation.Attempts = []*servicegenai.GenerationAttempt{
			{Number: 1, Response: []byte(`{"status":"in_progress"}`)},
			{Number: 2, Response: []byte(response)},
		}
	}

	return generation
}

// cancelledGeneration returns a cancelled genai generation, with the error genai gives it.
func cancelledGeneration() *servicegenai.Generation {
	generation := genaiGeneration(servicegenai.GenerationStatusCancelled, "")
	generation.Error = "generation cancelled"

	return generation
}

// openaiModel names the model of the completed OpenAI responses.
const openaiModel = "gpt-6.1-terra-2026-09-30"

// openaiResponse returns a completed OpenAI Responses object of [openaiModel], under the output format named
// after a kind, whose message holds content.
func openaiResponse(kind string, content ...map[string]string) string {
	return mustJSON(map[string]any{
		"model":  openaiModel,
		"status": "completed",
		"text":   map[string]any{"format": map[string]string{"type": "json_schema", "name": kind}},
		"output": []any{
			map[string]string{"type": "reasoning"},
			map[string]any{"type": "message", "content": content},
		},
	})
}

// outputText is a content part of an OpenAI message.
func outputText(text string) map[string]string {
	return map[string]string{"type": "output_text", "text": text}
}
