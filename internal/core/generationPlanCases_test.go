package core_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
)

// generationPlanCase is one request of [TestGenerationPlan], with what its plan holds.
type generationPlanCase struct {
	name string

	request *core.GenerationPlanRequest
	// extra is the budget the case gives beyond what the frame, the task and the scenes of targets take.
	// Unbounded gives the case all the budget it needs instead.
	extra     int
	unbounded bool
	// limit caps the prompt below what the tests build, when set.
	limit int

	// expectLevels lists the level of each scene, in narrative order.
	expectLevels []string
	// expectContains and expectAbsent list text the user message holds, or lacks.
	expectContains []string
	expectAbsent   []string
	// expect checks the rest of the plan's one call.
	expect func(t *testing.T, call *core.ContextCall)

	expectErr error
}

// longText is a narration of about a thousand tokens, so a scene's manuscript outweighs its outline.
var longText = strings.Repeat("The sea rolled in. ", 220)

// longNotes holds notes of about 55 tokens, so an outline of [planProject] takes about 70.
var longNotes = strings.Repeat("Ana keeps the lighthouse. ", 8)

const (
	// outlineCost bounds the tokens an outline of [planProject] takes. Two outlines take more.
	outlineCost = 100
	// manuscriptCost bounds the tokens the manuscript of [planProject] adds to an outline.
	manuscriptCost = 1100
)

// planProject returns an English project of count scenes, each in a beat of its own. Each scene has a
// manuscript cell of one long narration block, and a notes cell.
func planProject(count int) *core.ProjectSnapshot {
	snapshot := &core.ProjectSnapshot{
		Project: &core.Project{ID: projectID, OwnerID: ownerID, Title: "Tides", Language: "en"},
		Layers: []*core.Layer{
			{ID: fixtureLayerID(0), ProjectID: projectID, Position: 0, Kind: core.LayerKindManuscript, Name: "Text"},
			{ID: fixtureLayerID(1), ProjectID: projectID, Position: 1, Kind: core.LayerKindNotes, Name: "Notes"},
		},
	}

	for position := range count {
		snapshot.Beats = append(snapshot.Beats, &core.Beat{
			ID: fixtureBeatID(position), ProjectID: projectID, Position: position,
			Title: "Beat " + strconv.Itoa(position+1), Color: beatColor, UpdatedAt: updatedAt,
		})
		snapshot.Scenes = append(snapshot.Scenes, &core.Scene{
			ID: fixtureSceneID(position), ProjectID: projectID, BeatID: fixtureBeatID(position),
			Title: "Scene " + strconv.Itoa(position+1), UpdatedAt: updatedAt,
		})
		snapshot.Cells = append(snapshot.Cells,
			planCell(position, 0, blocks(narration(longText))),
			planCell(position, 1, mustJSON(map[string]string{"text": "Notes " + strconv.Itoa(position+1) + ". " + longNotes})),
		)
	}

	return snapshot
}

// inOneBeat moves the scenes of the snapshot into its first beat, keeping their order, and removes the
// other beats.
func inOneBeat(snapshot *core.ProjectSnapshot) *core.ProjectSnapshot {
	for position, scene := range snapshot.Scenes {
		scene.BeatID, scene.Position = fixtureBeatID(0), position
	}

	snapshot.Beats = snapshot.Beats[:1]

	return snapshot
}

// planCell returns the cell of the scene and the layer at positions, holding content.
func planCell(scene, layer int, content string) *core.Cell {
	return &core.Cell{
		ProjectID: projectID, SceneID: fixtureSceneID(scene), LayerID: fixtureLayerID(layer),
		Content: json.RawMessage(content), UpdatedAt: updatedAt,
	}
}

// blocks returns manuscript content holding blocks.
func blocks(blocks ...string) string {
	return `{"blocks":[` + strings.Join(blocks, ",") + `]}`
}

func narration(text string) string {
	return mustJSON(map[string]string{"kind": "narration", "text": text})
}

func dialogue(speaker, text string) string {
	return mustJSON(map[string]any{
		"kind": "dialogue", "speaker": speaker, "parts": []any{map[string]string{"text": text}},
	})
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return string(data)
}

// setManuscript replaces the manuscript of the scene at position.
func setManuscript(snapshot *core.ProjectSnapshot, scene int, content string) *core.ProjectSnapshot {
	for _, cell := range snapshot.Cells {
		if cell.SceneID == fixtureSceneID(scene) && cell.LayerID == fixtureLayerID(0) {
			cell.Content = json.RawMessage(content)
		}
	}

	return snapshot
}

// addRecap gives the manuscript cell of the second scene a recap, which read the cell shift away from its
// UpdatedAt. A recap is fresh only when shift is zero.
func addRecap(snapshot *core.ProjectSnapshot, shift time.Duration) *core.ProjectSnapshot {
	snapshot.Recaps = append(snapshot.Recaps, &core.CellRecap{
		Recap:   recapOf("The tide takes the boat."),
		SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(0), CellUpdatedAt: updatedAt.Add(shift),
	})

	return snapshot
}

// recapOf returns a recap whose versions repeat text one, three and nine times, from essentials up, and
// which names Ana.
func recapOf(text string) core.Recap {
	return core.Recap{
		Condensed:  strings.Repeat(text+" ", 9),
		Compact:    strings.Repeat(text+" ", 3),
		Essentials: text,
		Names:      []string{"Ana"},
	}
}

// sizedRecap gives the manuscript of the second scene a fresh recap of about 500, 200 and 20 tokens.
func sizedRecap(snapshot *core.ProjectSnapshot) *core.ProjectSnapshot {
	snapshot.Recaps = append(snapshot.Recaps, &core.CellRecap{
		Recap: core.Recap{
			Condensed:  strings.Repeat("Waves. ", 285),
			Compact:    strings.Repeat("Tide. ", 133),
			Essentials: strings.Repeat("Gull. ", 13),
		},
		SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(0), CellUpdatedAt: updatedAt,
	})

	return snapshot
}

// longBeats gives every beat but the first a description of about 1,500 tokens and a fresh recap whose
// compact version takes about 300. The beat at stale holds a recap that read the beat before a change.
func longBeats(snapshot *core.ProjectSnapshot, stale int) *core.ProjectSnapshot {
	for position, beat := range snapshot.Beats[1:] {
		beat.Description = strings.Repeat("The harbor wakes. ", 333)

		read := beat.UpdatedAt
		if position+1 == stale {
			read = beat.UpdatedAt.Add(-time.Hour)
		}

		var sources []*core.BeatRecapSource

		for _, scene := range snapshot.Scenes {
			if scene.BeatID == beat.ID {
				sources = append(sources, &core.BeatRecapSource{SceneID: scene.ID, CellUpdatedAt: updatedAt})
			}
		}

		snapshot.BeatRecaps = append(snapshot.BeatRecaps, &core.BeatRecap{
			Recap: core.Recap{
				Compact: strings.Repeat("The boats come back. ", 57), Essentials: "The boats return.",
			},
			BeatID: beat.ID, BeatUpdatedAt: read, Sources: sources,
		})
	}

	return snapshot
}

// shortScenes gives every scene a manuscript of one short block.
func shortScenes(snapshot *core.ProjectSnapshot) *core.ProjectSnapshot {
	for position := range snapshot.Scenes {
		setManuscript(snapshot, position, blocks(narration("Dawn.")))
	}

	return snapshot
}

// beatLevels returns the level of each beat of a call, in order.
func beatLevels(call *core.ContextCall) []string {
	return lo.Map(call.Beats, func(beat *core.ContextBeat, _ int) string { return beat.Level })
}

func suggestRequest(snapshot *core.ProjectSnapshot, targets ...*core.GenerationTarget) *core.GenerationPlanRequest {
	return &core.GenerationPlanRequest{Snapshot: snapshot, Kind: core.GenerationKindSuggest, Targets: targets}
}

func cellTarget(scene, layer int, mode string) *core.GenerationTarget {
	return &core.GenerationTarget{SceneID: fixtureSceneID(scene), LayerID: fixtureLayerID(layer), Mode: mode}
}

func fieldTarget(scene int, field, mode string) *core.GenerationTarget {
	return &core.GenerationTarget{SceneID: fixtureSceneID(scene), Field: field, Mode: mode}
}

// with returns value once change has run on it.
func with[T any](value T, change func(T)) T {
	change(value)

	return value
}

// contextBlocks is manuscript content of nine blocks of about a hundred tokens each, numbered from 0.
var contextBlocks = blocks(lo.Times(9, func(index int) string {
	return narration("Block " + strconv.Itoa(index) + ". " + strings.Repeat("Waves. ", 56))
})...)

// fourBlocks is manuscript content of four short blocks, two of them dialogue.
var fourBlocks = blocks(
	narration("Dawn."), dialogue("Ana", "You came."), narration("Silence."), dialogue("Leo", "I did."),
)

func generationPlanCases() []*generationPlanCase {
	full, condensed, outline, omitted := core.ContextLevelFull, core.ContextLevelCondensed,
		core.ContextLevelOutline, core.ContextLevelOmitted
	compact, essentials := core.ContextLevelCompact, core.ContextLevelEssentials
	write, revise := core.TargetModeWrite, core.TargetModeRevise

	return []*generationPlanCase{
		{
			name:         "Success/Unbounded",
			request:      suggestRequest(planProject(4), cellTarget(0, 0, revise)),
			unbounded:    true,
			expectLevels: []string{full, full, full, full},
		},
		{
			name:         "Success/NoBudget",
			request:      suggestRequest(planProject(4), cellTarget(0, 0, revise)),
			expectLevels: []string{full, omitted, omitted, omitted},
			expectAbsent: []string{`<scene handle="S2" level=`},
		},
		{
			name:         "Success/OutlinesFirst",
			request:      suggestRequest(planProject(4), cellTarget(0, 0, revise)),
			extra:        3*outlineCost + manuscriptCost/2,
			expectLevels: []string{full, outline, outline, outline},
		},
		{
			name:         "Success/FarthestDegradeFirst",
			request:      suggestRequest(planProject(4), cellTarget(0, 0, revise)),
			extra:        3*outlineCost + 2*manuscriptCost,
			expectLevels: []string{full, full, full, outline},
		},
		{
			name:         "Success/FarthestOmittedFirst",
			request:      suggestRequest(planProject(4), cellTarget(3, 0, revise)),
			extra:        outlineCost + outlineCost/10,
			expectLevels: []string{omitted, omitted, outline, full},
		},
		{
			// The plan stays within the budget while hundreds of scenes share it.
			name:    "Success/ManyScenes",
			request: suggestRequest(planProject(200), cellTarget(0, 0, revise)),
			extra:   100 * outlineCost,
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()

				outlines := lo.CountBy(call.Scenes, func(scene *core.ContextScene) bool { return scene.Level == outline })
				require.Greater(t, outlines, 100)
				require.Equal(t, omitted, call.Scenes[199].Level)
			},
		},
		{
			name: "Success/SharedNames",
			request: suggestRequest(
				setManuscript(setManuscript(planProject(4), 0, blocks(dialogue("Élodie", "Here."), dialogue("L'Inconnu", "No."))),
					3, blocks(narration(longText), dialogue("elodie", "Again."), dialogue("Inconnu", "Still."))),
				cellTarget(0, 0, revise),
			),
			extra:        3*outlineCost + manuscriptCost + manuscriptCost/2,
			expectLevels: []string{full, outline, outline, full},
		},
		{
			name:           "Success/Recap",
			request:        suggestRequest(addRecap(planProject(4), 0), cellTarget(0, 0, revise)),
			extra:          3*outlineCost + manuscriptCost/2,
			expectLevels:   []string{full, condensed, outline, outline},
			expectContains: []string{`<recap layer="L1">The tide takes the boat. The tide takes the boat.`},
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.Equal(t, []*core.ContextCell{
					{LayerID: fixtureLayerID(0), Recap: condensed, UpdatedAt: updatedAt},
					{LayerID: fixtureLayerID(1), UpdatedAt: updatedAt},
				}, call.Scenes[1].Cells)
			},
		},
		{
			name:         "Success/Condensed",
			request:      suggestRequest(sizedRecap(planProject(2)), cellTarget(0, 0, revise)),
			extra:        outlineCost + 600,
			expectLevels: []string{full, condensed},
		},
		{
			name:         "Success/Compact",
			request:      suggestRequest(sizedRecap(planProject(2)), cellTarget(0, 0, revise)),
			extra:        outlineCost + 300,
			expectLevels: []string{full, compact},
		},
		{
			name:           "Success/Essentials",
			request:        suggestRequest(sizedRecap(planProject(2)), cellTarget(0, 0, revise)),
			extra:          outlineCost + 100,
			expectLevels:   []string{full, essentials},
			expectContains: []string{`<scene handle="S2" level="essentials">` + "\n" + `<recap layer="L1">Gull. Gull.`},
		},
		{
			// The beats farthest from the target degrade first: whole, then compact, then essentials.
			name:         "Success/FrameCollapse",
			request:      suggestRequest(longBeats(shortScenes(planProject(5)), 0), cellTarget(0, 0, revise)),
			extra:        2200,
			expectAbsent: []string{`<beat title="Beat 1" level=`},
			expectContains: []string{
				`<beat title="Beat 4" level="essentials" scenes="S4">The boats return.</beat>`,
				`<beat title="Beat 5" level="essentials" scenes="S5">The boats return.</beat>`,
			},
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.Equal(t, []string{full, full, compact, essentials, essentials}, beatLevels(call))
			},
		},
		{
			name:           "Success/FrameStaleBeatRecap",
			request:        suggestRequest(longBeats(shortScenes(planProject(5)), 2), cellTarget(0, 0, revise)),
			extra:          2200,
			expectContains: []string{`<beat title="Beat 3" level="outline" scenes="S3"/>`},
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.Equal(t, []string{full, full, outline, compact, essentials}, beatLevels(call))
			},
		},
		{
			name:      "Success/FrameWhole",
			request:   suggestRequest(longBeats(shortScenes(planProject(3)), 0), cellTarget(0, 0, revise)),
			unbounded: true,
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.Equal(t, []string{full, full, full}, beatLevels(call))
			},
		},
		{
			name:         "Success/StaleRecap",
			request:      suggestRequest(addRecap(planProject(4), -time.Hour), cellTarget(0, 0, revise)),
			extra:        3*outlineCost + manuscriptCost/2,
			expectLevels: []string{full, outline, outline, outline},
			expectAbsent: []string{"The tide takes the boat."},
		},
		{
			name:         "Success/StaleRecapOfAnotherWrite",
			request:      suggestRequest(addRecap(planProject(4), time.Hour), cellTarget(0, 0, revise)),
			extra:        3*outlineCost + manuscriptCost/2,
			expectLevels: []string{full, outline, outline, outline},
			expectAbsent: []string{"The tide takes the boat."},
		},
		{
			name: "Success/OverrideStalesRecap",
			request: with(suggestRequest(addRecap(planProject(4), 0), cellTarget(0, 0, revise)),
				func(request *core.GenerationPlanRequest) {
					request.Overrides = &core.GenerationOverrides{Cells: []*core.CellOverride{{
						SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(0),
						Content: json.RawMessage(blocks(narration(longText))),
					}}}
				}),
			extra:        3*outlineCost + manuscriptCost/2,
			expectLevels: []string{full, outline, outline, outline},
			expectAbsent: []string{"The tide takes the boat."},
		},
		{
			name: "Success/ExcludedLayer",
			request: with(suggestRequest(planProject(3), cellTarget(0, 0, revise)), func(request *core.GenerationPlanRequest) {
				request.Exclude = []uuid.UUID{fixtureLayerID(1)}
			}),
			unbounded:    true,
			expectAbsent: []string{"Notes 1", "Notes 2", `name="Notes"`},
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.Equal(t, []*core.ContextLayer{{Handle: "L1", LayerID: fixtureLayerID(0)}}, call.Layers)
				require.Equal(t, []*core.ContextCell{{LayerID: fixtureLayerID(0), UpdatedAt: updatedAt}}, call.Scenes[1].Cells)
			},
		},
		{
			name: "Success/ExcludedTargetLayer",
			request: with(suggestRequest(planProject(3), cellTarget(0, 1, revise)), func(request *core.GenerationPlanRequest) {
				request.Exclude = []uuid.UUID{fixtureLayerID(1)}
			}),
			unbounded: true,
			expectContains: []string{
				`<layer handle="L2" kind="notes" name="Notes">`,
				`<target handle="T1" scene="S1" layer="L2" mode="revise">` + "\n<text>Notes 1. ",
			},
			expectAbsent: []string{"Notes 2", "Notes 3"},
		},
		{
			// Four blocks of context fit, two on each side, before any other scene.
			name: "Success/TargetContextNearestFirst",
			request: suggestRequest(setManuscript(planProject(2), 0, contextBlocks), with(cellTarget(0, 0, revise),
				func(target *core.GenerationTarget) {
					target.Range = &core.Range{Start: 4, End: 5}
				})),
			extra:        450,
			expectLevels: []string{full, omitted},
			expectContains: []string{
				`<before omitted="2">` + "\n<narration>Block 2. ",
				"<narration>Block 3. ",
				`<after omitted="2">` + "\n<narration>Block 5. ",
				"<narration>Block 6. ",
			},
			expectAbsent: []string{"Block 1. ", "Block 7. "},
		},
		{
			name: "Success/TargetContextOmitted",
			request: suggestRequest(setManuscript(planProject(1), 0, contextBlocks), with(cellTarget(0, 0, revise),
				func(target *core.GenerationTarget) {
					target.Range = &core.Range{Start: 4, End: 5}
				})),
			expectContains: []string{`<before omitted="4">` + "\n</before>", `<after omitted="4">` + "\n</after>"},
		},
		{
			name: "Success/Overrides",
			request: with(suggestRequest(planProject(3), cellTarget(0, 0, revise)), func(request *core.GenerationPlanRequest) {
				request.Overrides = &core.GenerationOverrides{
					Scenes: []*core.SceneOverride{{ID: fixtureSceneID(1), Title: "Unsaved title", Description: "Unsaved"}},
					Cells: []*core.CellOverride{{
						SceneID: fixtureSceneID(2), LayerID: fixtureLayerID(1), Content: json.RawMessage(`{"text":"Unsaved notes"}`),
					}},
				}
			}),
			unbounded:      true,
			expectContains: []string{`<scene handle="S2" title="Unsaved title">Unsaved</scene>`, "Unsaved notes"},
			expectAbsent:   []string{"Scene 2", "Notes 3"},
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.True(t, call.Scenes[1].Override)
				require.Zero(t, call.Scenes[1].UpdatedAt)
				require.False(t, call.Scenes[2].Override)
				require.Equal(t, &core.ContextCell{LayerID: fixtureLayerID(1), Override: true}, call.Scenes[2].Cells[1])
			},
		},
		{
			name: "Success/ReviseRange",
			request: suggestRequest(setManuscript(planProject(2), 0, fourBlocks), with(cellTarget(0, 0, revise),
				func(target *core.GenerationTarget) {
					target.Range, target.Kept = &core.Range{Start: 1, End: 3}, []int{2}
				})),
			unbounded: true,
			expectContains: []string{
				"<before>\n<narration>Dawn.</narration>\n</before>",
				"<range>\n" + `<dialogue handle="B1" speaker="Ana"><text>You came.</text></dialogue>` + "\n" +
					`<narration handle="B2" kept="true">Silence.</narration>` + "\n</range>",
				"<after>\n" + `<dialogue speaker="Leo"><text>I did.</text></dialogue>` + "\n</after>",
			},
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.Equal(t, []*core.ContextBlock{{Handle: "B1", Position: 1}, {Handle: "B2", Position: 2}},
					call.Targets[0].Blocks)
				require.Equal(t, &core.Range{Start: 1, End: 3}, call.Targets[0].Range)
				require.Equal(t, []int{2}, call.Targets[0].Kept)
			},
		},
		{
			name: "Success/WriteHidesReplacedContent",
			request: suggestRequest(setManuscript(planProject(2), 0, fourBlocks), with(cellTarget(0, 0, write),
				func(target *core.GenerationTarget) {
					target.Kept = []int{1}
				})),
			unbounded: true,
			expectContains: []string{
				"<range>\n" + `<dialogue handle="B1" kept="true" speaker="Ana"><text>You came.</text></dialogue>` +
					"\n</range>",
			},
			expectAbsent: []string{"Dawn.", "Silence.", "I did."},
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.Equal(t, []*core.ContextBlock{{Handle: "B1", Position: 1}}, call.Targets[0].Blocks)
				require.Equal(t, &core.Range{Start: 0, End: 4}, call.Targets[0].Range)
			},
		},
		{
			name: "Success/TextTargets",
			request: suggestRequest(planProject(3),
				fieldTarget(0, core.SceneFieldTitle, revise),
				fieldTarget(1, core.SceneFieldDescription, write),
				cellTarget(2, 1, revise),
				cellTarget(1, 1, write),
			),
			unbounded: true,
			expectContains: []string{
				`<target handle="T1" scene="S1" field="title" mode="revise">` + "\n<text>Scene 1</text>\n</target>",
				`<target handle="T2" scene="S2" field="description" mode="write"/>`,
				`<target handle="T3" scene="S3" layer="L2" mode="revise">` + "\n<text>Notes 3. ",
				`<target handle="T4" scene="S2" layer="L2" mode="write"/>`,
				`<cell layer="L2" target="T4"/>`,
			},
			expectAbsent: []string{"Notes 2"},
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.Equal(t, &core.ContextTarget{
					Handle: "T1", SceneID: fixtureSceneID(0), Field: core.SceneFieldTitle, Mode: revise, UpdatedAt: updatedAt,
				}, call.Targets[0])
				require.Equal(t, &core.ContextTarget{
					Handle: "T4", SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(1), Mode: write, UpdatedAt: updatedAt,
				}, call.Targets[3])
			},
		},
		{
			name: "Success/WriteHidesTitle",
			request: suggestRequest(setManuscript(planProject(2), 0, blocks(narration("Dawn."))),
				fieldTarget(1, core.SceneFieldTitle, write)),
			unbounded:      true,
			expectContains: []string{`<scene handle="S2"/>`},
			expectAbsent:   []string{"Scene 2"},
		},
		{
			name: "Success/Handles",
			request: suggestRequest(setManuscript(setManuscript(planProject(3), 0, fourBlocks), 2, fourBlocks),
				cellTarget(0, 0, revise),
				with(cellTarget(2, 0, revise), func(target *core.GenerationTarget) {
					target.Range = &core.Range{Start: 3, End: 4}
				}),
			),
			unbounded: true,
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()

				scene, ok := call.Scene("S3")
				require.True(t, ok)
				require.Equal(t, fixtureSceneID(2), scene.SceneID)

				_, ok = call.Scene("S4")
				require.False(t, ok)

				first, ok := call.Target("M1")
				require.True(t, ok)

				position, ok := first.Block("B4")
				require.True(t, ok)
				require.Equal(t, 3, position)

				second, ok := call.Target("M2")
				require.True(t, ok)

				position, ok = second.Block("B5")
				require.True(t, ok)
				require.Equal(t, 3, position)

				_, ok = second.Block("B1")
				require.False(t, ok, "a block of another target")

				_, ok = call.Target("M3")
				require.False(t, ok)
			},
		},
		{
			name: "Success/Consolidate",
			request: &core.GenerationPlanRequest{
				Snapshot: planProject(3), Kind: core.GenerationKindConsolidate,
				Cell: &core.GenerationCell{SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(0)},
			},
			unbounded:    true,
			expectLevels: []string{full, full, full},
			expectContains: []string{
				`<target handle="T1" scene="S2" layer="L1">` + "\n<narration>The sea", `<cell layer="L1" target="T1"/>`,
			},
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.Equal(t, []*core.ContextTarget{{
					Handle: "T1", SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(0), UpdatedAt: updatedAt,
				}}, call.Targets)
			},
		},
		{
			name: "Success/ConsolidateBeat",
			request: &core.GenerationPlanRequest{
				Snapshot: sizedRecap(inOneBeat(planProject(3))), Kind: core.GenerationKindConsolidate,
				BeatID: fixtureBeatID(0),
			},
			unbounded:    true,
			expectLevels: []string{full, full, full},
			expectContains: []string{
				`<target handle="T1" beat="Beat 1">` + "\n" + `<scene handle="S1">` + "\n<narration>The sea",
				`<scene handle="S2"><recap>Tide. Tide.`,
				`<cell layer="L1" target="T1"/>`,
			},
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.Equal(t, []*core.ContextTarget{
					{Handle: "T1", BeatID: fixtureBeatID(0), UpdatedAt: updatedAt},
				}, call.Targets)
			},
		},
		{
			name: "Success/Check",
			request: &core.GenerationPlanRequest{
				Snapshot: planProject(4), Kind: core.GenerationKindCheck, Scenes: &core.Range{Start: 1, End: 3},
				Instruction: "Look at the tides.",
			},
			extra:          4*outlineCost + 2*manuscriptCost,
			expectLevels:   []string{outline, full, full, outline},
			expectContains: []string{`<check from="S2" to="S3"/>`, "<instruction>Look at the tides.</instruction>"},
		},
		{
			name:           "Success/CheckAll",
			request:        &core.GenerationPlanRequest{Snapshot: planProject(2), Kind: core.GenerationKindCheck},
			unbounded:      true,
			expectLevels:   []string{full, full},
			expectContains: []string{`<check scenes="all"/>`},
		},
		{
			name: "Success/Escapes",
			request: suggestRequest(setManuscript(planProject(1), 0, blocks(narration(`<b> & "c"`))),
				cellTarget(0, 0, revise)),
			unbounded:      true,
			expectContains: []string{`<narration handle="B1">&lt;b&gt; &amp; &quot;c&quot;</narration>`},
		},
		{
			name:      "Error/ContextTooLarge",
			request:   suggestRequest(planProject(1), cellTarget(0, 0, revise)),
			limit:     100,
			expectErr: core.ErrContextTooLarge,
		},
		{
			// Beats come in slice order 1, 0, and the first beat holds two scenes listed out of order.
			name: "Success/NarrativeOrder",
			request: suggestRequest(with(inOneBeat(planProject(3)), func(snapshot *core.ProjectSnapshot) {
				snapshot.Beats = []*core.Beat{
					{ID: fixtureBeatID(1), ProjectID: projectID, Position: 1, Title: "Second", Color: beatColor},
					{ID: fixtureBeatID(0), ProjectID: projectID, Position: 0, Title: "First", Color: beatColor},
				}
				snapshot.Scenes[0].BeatID, snapshot.Scenes[0].Position = fixtureBeatID(1), 0
				snapshot.Scenes[1].Position, snapshot.Scenes[2].Position = 1, 0
			}), cellTarget(0, 0, revise)),
			unbounded: true,
			expectContains: []string{
				`<beat title="First">` + "\n" + `<scene handle="S1" title="Scene 3"/>` + "\n" +
					`<scene handle="S2" title="Scene 2"/>` + "\n</beat>",
				`<beat title="Second">` + "\n" + `<scene handle="S3" title="Scene 1"/>`,
			},
			expect: func(t *testing.T, call *core.ContextCall) {
				t.Helper()
				require.Equal(t, []uuid.UUID{fixtureSceneID(2), fixtureSceneID(1), fixtureSceneID(0)},
					lo.Map(call.Scenes, func(scene *core.ContextScene, _ int) uuid.UUID { return scene.SceneID }))
			},
		},
		{
			name: "Success/BeatPrompts",
			request: suggestRequest(with(planProject(2), func(snapshot *core.ProjectSnapshot) {
				snapshot.Beats[0].Description, snapshot.Beats[0].Prompt = "Ana returns.", "Keep it quiet."
				snapshot.Scenes[0].Prompt = "Ana speaks first."
			}), cellTarget(0, 0, revise)),
			unbounded: true,
			expectContains: []string{
				`<beat title="Beat 1">` + "\n<description>Ana returns.</description>",
				`<scene handle="S1" level="full">` + "\n<beat-prompt>Keep it quiet.</beat-prompt>\n" +
					"<prompt>Ana speaks first.</prompt>",
			},
		},
		{
			name: "Error/SceneBeatNotFound",
			request: suggestRequest(with(planProject(2), func(snapshot *core.ProjectSnapshot) {
				snapshot.Beats = snapshot.Beats[:1]
			}), cellTarget(0, 0, revise)),
			expectErr: core.ErrBeatNotFound,
		},
		{
			name:      "Error/NoTarget",
			request:   suggestRequest(planProject(2)),
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "Error/UnknownKind",
			request:   &core.GenerationPlanRequest{Snapshot: planProject(2), Kind: "rewrite"},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TargetsOfCheck",
			request: with(suggestRequest(planProject(2), cellTarget(0, 0, revise)), func(request *core.GenerationPlanRequest) {
				request.Kind = core.GenerationKindCheck
			}),
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ConsolidateInstruction",
			request: &core.GenerationPlanRequest{
				Snapshot: planProject(2), Kind: core.GenerationKindConsolidate, Instruction: "Shorter.",
				Cell: &core.GenerationCell{SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(0)},
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ConsolidateOverrides",
			request: &core.GenerationPlanRequest{
				Snapshot: planProject(2), Kind: core.GenerationKindConsolidate, Overrides: &core.GenerationOverrides{},
				Cell: &core.GenerationCell{SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(0)},
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ConsolidateCellAndBeat",
			request: &core.GenerationPlanRequest{
				Snapshot: planProject(2), Kind: core.GenerationKindConsolidate, BeatID: fixtureBeatID(0),
				Cell: &core.GenerationCell{SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(0)},
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "Error/ConsolidateNothing",
			request:   &core.GenerationPlanRequest{Snapshot: planProject(2), Kind: core.GenerationKindConsolidate},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ConsolidateBeatNotFound",
			request: &core.GenerationPlanRequest{
				Snapshot: planProject(2), Kind: core.GenerationKindConsolidate, BeatID: fixtureBeatID(7),
			},
			expectErr: core.ErrBeatNotFound,
		},
		{
			name: "Error/ConsolidateSceneNotFound",
			request: &core.GenerationPlanRequest{
				Snapshot: planProject(2), Kind: core.GenerationKindConsolidate,
				Cell: &core.GenerationCell{SceneID: fixtureSceneID(5), LayerID: fixtureLayerID(0)},
			},
			expectErr: core.ErrSceneNotFound,
		},
		{
			name:      "Error/UnknownMode",
			request:   suggestRequest(planProject(2), cellTarget(0, 0, "rewrite")),
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/FieldAndLayer",
			request: suggestRequest(planProject(2), with(cellTarget(0, 0, revise), func(target *core.GenerationTarget) {
				target.Field = core.SceneFieldTitle
			})),
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "Error/TargetSceneNotFound",
			request:   suggestRequest(planProject(2), cellTarget(5, 0, revise)),
			expectErr: core.ErrSceneNotFound,
		},
		{
			name:      "Error/TargetLayerNotFound",
			request:   suggestRequest(planProject(2), cellTarget(0, 5, revise)),
			expectErr: core.ErrLayerNotFound,
		},
		{
			name:      "Error/RepeatedTarget",
			request:   suggestRequest(planProject(2), cellTarget(0, 0, revise), cellTarget(0, 0, write)),
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/RangePastEnd",
			request: suggestRequest(planProject(2), with(cellTarget(0, 0, revise), func(target *core.GenerationTarget) {
				target.Range = &core.Range{Start: 0, End: 2}
			})),
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/KeptOutsideRange",
			request: suggestRequest(setManuscript(planProject(2), 0, fourBlocks), with(cellTarget(0, 0, revise),
				func(target *core.GenerationTarget) {
					target.Range, target.Kept = &core.Range{Start: 1, End: 2}, []int{2}
				})),
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/RangeOnTextTarget",
			request: suggestRequest(planProject(2), with(cellTarget(0, 1, revise), func(target *core.GenerationTarget) {
				target.Range = &core.Range{Start: 0, End: 0}
			})),
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ExcludedLayerNotFound",
			request: with(suggestRequest(planProject(2), cellTarget(0, 0, revise)), func(request *core.GenerationPlanRequest) {
				request.Exclude = []uuid.UUID{fixtureLayerID(5)}
			}),
			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/OverrideSceneNotFound",
			request: with(suggestRequest(planProject(2), cellTarget(0, 0, revise)), func(request *core.GenerationPlanRequest) {
				request.Overrides = &core.GenerationOverrides{Scenes: []*core.SceneOverride{{ID: fixtureSceneID(5), Title: "Lost"}}}
			}),
			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/OverrideInvalidContent",
			request: with(suggestRequest(planProject(2), cellTarget(0, 1, revise)), func(request *core.GenerationPlanRequest) {
				request.Overrides = &core.GenerationOverrides{Cells: []*core.CellOverride{{
					SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(0), Content: json.RawMessage(`{"text":"Not blocks"}`),
				}}}
			}),
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/RepeatedCellOverride",
			request: with(suggestRequest(planProject(2), cellTarget(0, 0, revise)), func(request *core.GenerationPlanRequest) {
				override := &core.CellOverride{
					SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(1), Content: json.RawMessage(`{"text":"Twice"}`),
				}
				request.Overrides = &core.GenerationOverrides{Cells: []*core.CellOverride{override, override}}
			}),
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/CheckPastEnd",
			request: &core.GenerationPlanRequest{
				Snapshot: planProject(2), Kind: core.GenerationKindCheck, Scenes: &core.Range{Start: 1, End: 3},
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/CheckEmptyRange",
			request: &core.GenerationPlanRequest{
				Snapshot: planProject(2), Kind: core.GenerationKindCheck, Scenes: &core.Range{Start: 1, End: 1},
			},
			expectErr: core.ErrInvalidRequest,
		},
	}
}

// generationSplitCase is one request of [TestGenerationPlan] that may outgrow a single call, with the
// settings of its kind.
type generationSplitCase struct {
	name string

	request *core.GenerationPlanRequest
	// settings returns the settings of the case's kind from single, the estimate of the request as one call
	// with its frame collapsed and no context.
	settings func(single int) config.GenerationKind

	// expect checks the plan.
	expect func(t *testing.T, plan *core.ContextPlan)

	expectErr error
}

// splitSettings returns settings with neither limit nor output cap in the way, and a budget that fits only
// the parts that always enter.
func splitSettings() config.GenerationKind {
	return config.GenerationKind{Model: "a-model", Reasoning: "low", MaxOutputTokens: 1 << 20, Limit: 1 << 30}
}

// spansProject is a [planProject] of three scenes. The first holds [contextBlocks], and the third two
// short blocks.
func spansProject() *core.ProjectSnapshot {
	return setManuscript(setManuscript(planProject(3), 0, contextBlocks), 2,
		blocks(narration("Dawn."), narration("Dusk.")))
}

// spansRequest revises the manuscript of each scene of [spansProject], keeping the fourth block of the
// first, and the title of the second.
func spansRequest() *core.GenerationPlanRequest {
	return suggestRequest(spansProject(),
		with(cellTarget(0, 0, core.TargetModeRevise), func(target *core.GenerationTarget) { target.Kept = []int{3} }),
		fieldTarget(1, core.SceneFieldTitle, core.TargetModeRevise),
		cellTarget(2, 0, core.TargetModeRevise),
	)
}

// checkRequest checks every scene of a project of six.
func checkRequest() *core.GenerationPlanRequest {
	return &core.GenerationPlanRequest{Snapshot: planProject(6), Kind: core.GenerationKindCheck}
}

// checkSettings gives a check a budget whose window share holds two scenes whole, and the budget not all
// six.
func checkSettings(split bool) func(int) config.GenerationKind {
	return func(int) config.GenerationKind {
		settings := splitSettings()
		settings.Budget, settings.Split = 6000, split

		return settings
	}
}

func generationPlanSplitCases() []*generationSplitCase {
	const full = core.ContextLevelFull

	return []*generationSplitCase{
		{
			name: "Success/LongContext",
			request: suggestRequest(planProject(2),
				cellTarget(0, 0, core.TargetModeRevise), cellTarget(1, 0, core.TargetModeRevise)),
			settings: func(single int) config.GenerationKind {
				settings := splitSettings()
				settings.Limit, settings.LongLimit = single-1, single

				return settings
			},
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Len(t, plan.Calls, 1)
				require.Len(t, plan.Calls[0].Targets, 2)
			},
		},
		{
			name: "Error/LongContextOff",
			request: suggestRequest(planProject(2),
				cellTarget(0, 0, core.TargetModeRevise), cellTarget(1, 0, core.TargetModeRevise)),
			settings: func(single int) config.GenerationKind {
				settings := splitSettings()
				settings.Limit = single - 1

				return settings
			},
			expectErr: core.ErrContextTooLarge,
		},
		{
			name: "Error/PastLongLimit",
			request: suggestRequest(planProject(2),
				cellTarget(0, 0, core.TargetModeRevise), cellTarget(1, 0, core.TargetModeRevise)),
			settings: func(single int) config.GenerationKind {
				settings := splitSettings()
				settings.Limit, settings.LongLimit = single-2, single-1

				return settings
			},
			expectErr: core.ErrContextTooLarge,
		},
		{
			// The targets together pass the limit, and each fits alone.
			name: "Success/SplitTargets",
			request: suggestRequest(planProject(3),
				cellTarget(0, 0, core.TargetModeRevise), fieldTarget(1, core.SceneFieldTitle, core.TargetModeWrite),
				cellTarget(2, 0, core.TargetModeRevise)),
			settings: func(single int) config.GenerationKind {
				settings := splitSettings()
				settings.Limit, settings.Split = single-manuscriptCost/2, true

				return settings
			},
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Equal(t, []string{"M1", "T1", "M2"}, callTargets(plan))

				for _, call := range plan.Calls {
					require.Less(t, call.Estimate, plan.Calls[0].Estimate+manuscriptCost)
				}

				require.Equal(t, []*core.ContextBlock{{Handle: "B2", Position: 0}}, plan.Calls[2].Targets[0].Blocks)
			},
		},
		{
			name: "Error/SplitOff",
			request: suggestRequest(planProject(3),
				cellTarget(0, 0, core.TargetModeRevise), cellTarget(2, 0, core.TargetModeRevise)),
			settings: func(single int) config.GenerationKind {
				settings := splitSettings()
				settings.Limit = single - manuscriptCost/2

				return settings
			},
			expectErr: core.ErrContextTooLarge,
		},
		{
			// The first target's rewrite passes half the output cap, so its range takes four spans of two
			// rewritten blocks each at most. The kept block stays with its span.
			name:    "Success/Spans",
			request: spansRequest(),
			settings: func(int) config.GenerationKind {
				settings := splitSettings()
				settings.MaxOutputTokens, settings.Split = 500, true

				return settings
			},
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Equal(t, []string{"M1", "M1", "M1", "M1", "T1", "M2"}, callTargets(plan))

				spans := lo.Map(plan.Calls[:4], func(call *core.ContextCall, _ int) *core.Range {
					return call.Targets[0].Range
				})
				require.Equal(t, []*core.Range{{Start: 0, End: 2}, {Start: 2, End: 5}, {Start: 5, End: 7}, {Start: 7, End: 9}},
					spans)
				require.Equal(t, []int{3}, plan.Calls[1].Targets[0].Kept)
				require.Empty(t, plan.Calls[2].Targets[0].Kept)
				require.Equal(t, []*core.ContextBlock{{Handle: "B3", Position: 2}, {Handle: "B4", Position: 3}, {
					Handle: "B5", Position: 4,
				}}, plan.Calls[1].Targets[0].Blocks)
				require.Equal(t, []*core.ContextBlock{{Handle: "B10", Position: 0}, {Handle: "B11", Position: 1}},
					plan.Calls[5].Targets[0].Blocks)
				require.NotContains(t, plan.Calls[0].Prompt.Task, `handle="B3"`, "a block of another span")
			},
		},
		{
			name:    "Error/SpansOff",
			request: spansRequest(),
			settings: func(int) config.GenerationKind {
				settings := splitSettings()
				settings.MaxOutputTokens = 500

				return settings
			},
			expectErr: core.ErrContextTooLarge,
		},
		{
			// A single block takes the whole output cap, past half of it.
			name:    "Success/SingleBlock",
			request: suggestRequest(planProject(1), cellTarget(0, 0, core.TargetModeRevise)),
			settings: func(int) config.GenerationKind {
				settings := splitSettings()
				settings.MaxOutputTokens, settings.Split = manuscriptCost, true

				return settings
			},
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Len(t, plan.Calls, 1)
			},
		},
		{
			name:    "Error/SingleBlockPastCap",
			request: suggestRequest(planProject(1), cellTarget(0, 0, core.TargetModeRevise)),
			settings: func(int) config.GenerationKind {
				settings := splitSettings()
				settings.MaxOutputTokens, settings.Split = 1000, true

				return settings
			},
			expectErr: core.ErrContextTooLarge,
		},
		{
			// A write rewrites nothing the planner measures.
			name:    "Success/Write",
			request: suggestRequest(spansProject(), cellTarget(0, 0, core.TargetModeWrite)),
			settings: func(int) config.GenerationKind {
				settings := splitSettings()
				settings.MaxOutputTokens = 1

				return settings
			},
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Len(t, plan.Calls, 1)
			},
		},
		{
			name:     "Success/Windows",
			request:  checkRequest(),
			settings: checkSettings(true),
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Len(t, plan.Calls, 3)

				for index, call := range plan.Calls {
					require.LessOrEqual(t, call.Estimate, plan.Budget)
					require.Equal(t, full, call.Scenes[2*index].Level)
					require.Equal(t, full, call.Scenes[2*index+1].Level)
					require.Contains(t, call.Prompt.Task,
						fmt.Sprintf(`<check from="S%d" to="S%d"/>`, 2*index+1, 2*index+2))
					require.NotContains(t, call.Prompt.Neighborhood, "<prompt>", "a check shows no prompts")
				}
			},
		},
		{
			name:     "Success/WindowsOff",
			request:  checkRequest(),
			settings: checkSettings(false),
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Len(t, plan.Calls, 1)
				require.True(t, lo.ContainsBy(plan.Calls[0].Scenes, func(scene *core.ContextScene) bool {
					return scene.Level != full
				}))
			},
		},
		{
			name: "Error/ConsolidateNeverSplits",
			request: &core.GenerationPlanRequest{
				Snapshot: planProject(2), Kind: core.GenerationKindConsolidate,
				Cell: &core.GenerationCell{SceneID: fixtureSceneID(0), LayerID: fixtureLayerID(0)},
			},
			settings: func(single int) config.GenerationKind {
				settings := splitSettings()
				settings.Limit, settings.Split = single-1, true

				return settings
			},
			expectErr: core.ErrContextTooLarge,
		},
	}
}

// callTargets returns the handle of each call's targets, in call order.
func callTargets(plan *core.ContextPlan) []string {
	return lo.FlatMap(plan.Calls, func(call *core.ContextCall, _ int) []string {
		return lo.Map(call.Targets, func(target *core.ContextTarget, _ int) string { return target.Handle })
	})
}
