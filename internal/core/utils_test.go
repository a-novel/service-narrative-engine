package core_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"

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
	editedAt  = time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)
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

// lockRequest matches the lock of the test project by its owner, which stamps the edit at an instant the
// service captures.
var lockRequest = mock.MatchedBy(func(request *dao.ProjectLockRequest) bool {
	return request.ID == projectID && request.OwnerID == ownerID && recent(request.Now)
})

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
	return daoLayer(id, position, core.LayerKindNotes)
}

func daoProject() *dao.Project {
	return &dao.Project{
		ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
		CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt, LogoUpdatedAt: lo.ToPtr(editedAt),
	}
}

// coreProject returns the project [daoProject] reads.
func coreProject() *core.Project {
	return &core.Project{
		ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
		CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt, LogoVersion: lo.ToPtr(editedAt),
	}
}

// newJPEG encodes a gray gradient of the given size as a JPEG.
func newJPEG(width, height int) []byte {
	img := image.NewGray(image.Rect(0, 0, width, height))

	for x := range width {
		for y := range height {
			img.SetGray(x, y, color.Gray{Y: uint8((x + y) % 256)})
		}
	}

	var buf bytes.Buffer

	err := jpeg.Encode(&buf, img, nil)
	if err != nil {
		panic(err)
	}

	return buf.Bytes()
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

var (
	// withinBudget is a project at its byte budget, which a write may reach.
	withinBudget = &daoMock[*dao.ProjectBytes]{resp: &dao.ProjectBytes{StoredBytes: core.ProjectBytesMax}}
	// overBudget is a project one byte past its budget, which refuses the write that brought it there.
	overBudget = &daoMock[*dao.ProjectBytes]{resp: &dao.ProjectBytes{StoredBytes: core.ProjectBytesMax + 1}}
)

// holdGate takes the only slot of gate, as another read of a whole project would, until the returned
// function runs.
func holdGate(t *testing.T, gate *core.ProjectReadGate) func() {
	t.Helper()

	holding, release := make(chan struct{}), make(chan struct{})

	go func() {
		// The read returns nil, and the test only needs the slot it holds.
		_ = gate.Run(t.Context(), func(context.Context) error {
			close(holding)
			<-release

			return nil
		})
	}()

	<-holding

	return func() { close(release) }
}
