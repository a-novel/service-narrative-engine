package handlers_test

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

var (
	ownerID   = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	projectID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	beatID    = uuid.MustParse("00000000-0000-0000-0000-000000000031")
	sceneID   = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	layerID   = uuid.MustParse("00000000-0000-0000-0000-000000000021")

	createdAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt = time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)

	coreProject = &core.Project{
		ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Premise: "Marie attend.", Language: "fr",
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
	jsonProject = map[string]any{
		"id": projectID.String(), "title": "Les Heures claires", "premise": "Marie attend.", "language": "fr",
		"createdAt": "2021-01-01T00:00:00Z", "updatedAt": "2021-01-02T00:00:00Z",
	}

	coreBeat = &core.Beat{
		ID: beatID, ProjectID: projectID, Position: 1, Title: "Ouverture", Color: "#e0457b",
		Description: "Marie attend.", Prompt: "Lent.", UpdatedAt: updatedAt,
	}
	jsonBeat = map[string]any{
		"id": beatID.String(), "projectID": projectID.String(), "position": 1.0, "title": "Ouverture",
		"color": "#e0457b", "description": "Marie attend.", "prompt": "Lent.", "updatedAt": "2021-01-02T00:00:00Z",
	}

	coreScene = &core.Scene{
		ID: sceneID, ProjectID: projectID, BeatID: beatID, Position: 1, Title: "La fenêtre",
		Description: "Marie guette.", Prompt: "Lent.", UpdatedAt: updatedAt,
	}
	jsonScene = map[string]any{
		"id": sceneID.String(), "projectID": projectID.String(), "beatID": beatID.String(), "position": 1.0,
		"title": "La fenêtre", "description": "Marie guette.", "prompt": "Lent.", "updatedAt": "2021-01-02T00:00:00Z",
	}

	coreLayer = &core.Layer{
		ID: layerID, ProjectID: projectID, Position: 1, Kind: core.LayerKindNotes, Name: "Characters",
		Description: "Who appears", UpdatedAt: updatedAt,
	}
	jsonLayer = map[string]any{
		"id": layerID.String(), "projectID": projectID.String(), "position": 1.0, "kind": "notes",
		"name": "Characters", "description": "Who appears", "updatedAt": "2021-01-02T00:00:00Z",
	}

	coreCell = &core.Cell{
		ProjectID: projectID, SceneID: sceneID, LayerID: layerID, Content: json.RawMessage(`{"text":"Marie"}`),
		UpdatedAt: updatedAt,
	}
	coreWrittenCell = &core.WrittenCell{SceneID: sceneID, LayerID: layerID, UpdatedAt: updatedAt}
	jsonWrittenCell = map[string]any{
		"sceneID": sceneID.String(), "layerID": layerID.String(), "updatedAt": "2021-01-02T00:00:00Z",
	}

	jsonCell = map[string]any{
		"projectID": projectID.String(), "sceneID": sceneID.String(), "layerID": layerID.String(),
		"content":   map[string]any{"text": "Marie"},
		"updatedAt": "2021-01-02T00:00:00Z",
	}
)

const (
	// suggestOutput is a valid suggest output, with one text suggestion.
	suggestOutput = `{"manuscript":[],"text":[{"target":"T1","note":"Names the scene.","text":"La lettre"}]}`
	// suggestModel names the model that wrote [suggestOutput].
	suggestModel = "gpt-6.1-terra-2026-09-30"
)

var (
	// genaiID names the genai generation of the generation tests.
	genaiID = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")

	// jsonResult is the JSON of a succeeded genai generation of [suggestOutput].
	jsonResult = map[string]any{
		"id": genaiID.String(), "status": "succeeded", "output": mustDecode(suggestOutput), "model": suggestModel,
	}
)
