package core_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

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

	// expectLevels lists the level of each scene, in narrative order.
	expectLevels []string
	// expectContains and expectAbsent list text the user message holds, or lacks.
	expectContains []string
	expectAbsent   []string
	// expect checks the rest of the plan.
	expect func(t *testing.T, plan *core.ContextPlan)

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

// planProject returns an English project of count scenes. Each scene has a manuscript cell of one long
// narration block, and a notes cell.
func planProject(count int) *core.ProjectSnapshot {
	snapshot := &core.ProjectSnapshot{
		Project: &core.Project{ID: projectID, OwnerID: ownerID, Title: "Tides", Language: "en"},
		Layers: []*core.Layer{
			{ID: fixtureLayerID(0), ProjectID: projectID, Position: 0, Kind: core.LayerKindManuscript, Name: "Text"},
			{ID: fixtureLayerID(1), ProjectID: projectID, Position: 1, Kind: core.LayerKindNotes, Name: "Notes"},
		},
	}

	for position := range count {
		snapshot.Scenes = append(snapshot.Scenes, &core.Scene{
			ID: fixtureSceneID(position), ProjectID: projectID, Position: position,
			Title: "Scene " + strconv.Itoa(position+1), Color: sceneColor, UpdatedAt: updatedAt,
		})
		snapshot.Cells = append(snapshot.Cells,
			planCell(position, 0, blocks(narration(longText))),
			planCell(position, 1, mustJSON(map[string]string{"text": "Notes " + strconv.Itoa(position+1) + ". " + longNotes})),
		)
	}

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
		SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(0), Summary: "The tide takes the boat.",
		Names: []string{"Ana"}, CellUpdatedAt: updatedAt.Add(shift),
	})

	return snapshot
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

// fourBlocks is manuscript content of four short blocks, two of them dialogue.
var fourBlocks = blocks(
	narration("Dawn."), dialogue("Ana", "You came."), narration("Silence."), dialogue("Leo", "I did."),
)

func generationPlanCases() []*generationPlanCase {
	full, recap, outline, omitted := core.ContextLevelFull, core.ContextLevelRecap, core.ContextLevelOutline,
		core.ContextLevelOmitted
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
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()

				outlines := lo.CountBy(plan.Scenes, func(scene *core.ContextScene) bool { return scene.Level == outline })
				require.Greater(t, outlines, 100)
				require.Equal(t, omitted, plan.Scenes[199].Level)
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
			expectLevels:   []string{full, recap, outline, outline},
			expectContains: []string{`<recap layer="L1">The tide takes the boat.</recap>`},
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Equal(t, []*core.ContextCell{
					{LayerID: fixtureLayerID(0), Recap: true, UpdatedAt: updatedAt},
					{LayerID: fixtureLayerID(1), UpdatedAt: updatedAt},
				}, plan.Scenes[1].Cells)
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
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Equal(t, []*core.ContextLayer{{Handle: "L1", LayerID: fixtureLayerID(0)}}, plan.Layers)
				require.Equal(t, []*core.ContextCell{{LayerID: fixtureLayerID(0), UpdatedAt: updatedAt}}, plan.Scenes[1].Cells)
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
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.True(t, plan.Scenes[1].Override)
				require.Zero(t, plan.Scenes[1].UpdatedAt)
				require.False(t, plan.Scenes[2].Override)
				require.Equal(t, &core.ContextCell{LayerID: fixtureLayerID(1), Override: true}, plan.Scenes[2].Cells[1])
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
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Equal(t, []*core.ContextBlock{{Handle: "B1", Position: 1}, {Handle: "B2", Position: 2}},
					plan.Targets[0].Blocks)
				require.Equal(t, &core.Range{Start: 1, End: 3}, plan.Targets[0].Range)
				require.Equal(t, []int{2}, plan.Targets[0].Kept)
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
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Equal(t, []*core.ContextBlock{{Handle: "B1", Position: 1}}, plan.Targets[0].Blocks)
				require.Equal(t, &core.Range{Start: 0, End: 4}, plan.Targets[0].Range)
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
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Equal(t, &core.ContextTarget{
					Handle: "T1", SceneID: fixtureSceneID(0), Field: core.SceneFieldTitle, Mode: revise, UpdatedAt: updatedAt,
				}, plan.Targets[0])
				require.Equal(t, &core.ContextTarget{
					Handle: "T4", SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(1), Mode: write, UpdatedAt: updatedAt,
				}, plan.Targets[3])
			},
		},
		{
			name: "Success/WriteHidesTitle",
			request: suggestRequest(setManuscript(planProject(2), 0, blocks(narration("Dawn."))),
				fieldTarget(1, core.SceneFieldTitle, write)),
			unbounded:      true,
			expectContains: []string{`<scene handle="S2"></scene>`},
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
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()

				scene, ok := plan.Scene("S3")
				require.True(t, ok)
				require.Equal(t, fixtureSceneID(2), scene.SceneID)

				_, ok = plan.Scene("S4")
				require.False(t, ok)

				first, ok := plan.Target("T1")
				require.True(t, ok)

				position, ok := first.Block("B4")
				require.True(t, ok)
				require.Equal(t, 3, position)

				second, ok := plan.Target("T2")
				require.True(t, ok)

				position, ok = second.Block("B5")
				require.True(t, ok)
				require.Equal(t, 3, position)

				_, ok = second.Block("B1")
				require.False(t, ok, "a block of another target")

				_, ok = plan.Target("T3")
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
			expect: func(t *testing.T, plan *core.ContextPlan) {
				t.Helper()
				require.Equal(t, []*core.ContextTarget{{
					Handle: "T1", SceneID: fixtureSceneID(1), LayerID: fixtureLayerID(0), UpdatedAt: updatedAt,
				}}, plan.Targets)
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
