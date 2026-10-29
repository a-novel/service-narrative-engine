package core_test

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

// updateGolden rewrites the golden prompt files from the current templates: go test ./internal/core -update.
var updateGolden = flag.Bool("update", false, "rewrite the golden prompt files")

//go:embed testdata/prompts
var promptFixtures embed.FS

// promptProject is a project fixture of testdata/prompts, with a request of each generation kind.
// Scenes, layers and cells refer to each other by position.
type promptProject struct {
	Project struct {
		Title    string `yaml:"title"`
		Premise  string `yaml:"premise"`
		Language string `yaml:"language"`
	} `yaml:"project"`
	Layers []struct {
		Kind        string `yaml:"kind"`
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	} `yaml:"layers"`
	Beats []struct {
		Title       string         `yaml:"title"`
		Description string         `yaml:"description"`
		Prompt      string         `yaml:"prompt"`
		Recap       *promptRecap   `yaml:"recap"`
		Scenes      []*promptScene `yaml:"scenes"`
	} `yaml:"beats"`
	Budgets  map[string]int `yaml:"budgets"`
	Requests struct {
		Suggest struct {
			Targets []struct {
				Scene int         `yaml:"scene"`
				Layer *int        `yaml:"layer"`
				Field string      `yaml:"field"`
				Mode  string      `yaml:"mode"`
				Range *core.Range `yaml:"range"`
				Kept  []int       `yaml:"kept"`
			} `yaml:"targets"`
			Instruction string `yaml:"instruction"`
			Exclude     []int  `yaml:"exclude"`
			Overrides   struct {
				Scenes []struct {
					Scene       int    `yaml:"scene"`
					Title       string `yaml:"title"`
					Description string `yaml:"description"`
				} `yaml:"scenes"`
			} `yaml:"overrides"`
		} `yaml:"suggest"`
		Consolidate struct {
			Cell struct {
				Scene int `yaml:"scene"`
				Layer int `yaml:"layer"`
			} `yaml:"cell"`
		} `yaml:"consolidate"`
		ConsolidateBeat struct {
			Beat int `yaml:"beat"`
		} `yaml:"consolidateBeat"`
		Check struct {
			Scenes      *core.Range `yaml:"scenes"`
			Instruction string      `yaml:"instruction"`
		} `yaml:"check"`
	} `yaml:"requests"`
}

// promptScene is a scene of a [promptProject], with its cells.
type promptScene struct {
	Title       string        `yaml:"title"`
	Description string        `yaml:"description"`
	Prompt      string        `yaml:"prompt"`
	Cells       []*promptCell `yaml:"cells"`
}

// promptCell is a cell of a [promptScene], in the layer at a position.
type promptCell struct {
	Layer   int          `yaml:"layer"`
	Content any          `yaml:"content"`
	Recap   *promptRecap `yaml:"recap"`
}

// promptRecap is a recap of a [promptProject]. A stale recap read an older version of its source.
type promptRecap struct {
	Condensed  string   `yaml:"condensed"`
	Compact    string   `yaml:"compact"`
	Essentials string   `yaml:"essentials"`
	Names      []string `yaml:"names"`
	Stale      bool     `yaml:"stale"`
}

// recap returns the versions of the recap, and the instant it read its source at.
func (recap *promptRecap) recap() (core.Recap, time.Time) {
	read := updatedAt
	if recap.Stale {
		read = updatedAt.Add(-time.Hour)
	}

	return core.Recap{
		Condensed: recap.Condensed, Compact: recap.Compact, Essentials: recap.Essentials, Names: recap.Names,
	}, read
}

// fixtureBeatID gives the beat at a position of a fixture its ID.
func fixtureBeatID(position int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-00000003%04d", position))
}

// fixtureSceneID and fixtureLayerID give the scene at a narrative position and the layer at a position of a
// fixture their ID.
func fixtureSceneID(position int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-00000001%04d", position))
}

func fixtureLayerID(position int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-00000002%04d", position))
}

// loadPromptProject reads the fixture of a language, and returns its snapshot and budgets.
func loadPromptProject(t *testing.T, language string) (*promptProject, *core.ProjectSnapshot) {
	t.Helper()

	data, err := promptFixtures.ReadFile("testdata/prompts/project." + language + ".yaml")
	require.NoError(t, err)

	fixture := &promptProject{}
	require.NoError(t, yaml.Unmarshal(data, fixture))

	snapshot := &core.ProjectSnapshot{Project: &core.Project{
		ID: projectID, OwnerID: ownerID, Title: fixture.Project.Title, Premise: fixture.Project.Premise,
		Language: fixture.Project.Language, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}}

	for position, layer := range fixture.Layers {
		snapshot.Layers = append(snapshot.Layers, &core.Layer{
			ID: fixtureLayerID(position), ProjectID: projectID, Position: position, Kind: layer.Kind,
			Name: layer.Name, Description: layer.Description, UpdatedAt: updatedAt,
		})
	}

	var position int

	for beatPosition, beat := range fixture.Beats {
		snapshot.Beats = append(snapshot.Beats, &core.Beat{
			ID: fixtureBeatID(beatPosition), ProjectID: projectID, Position: beatPosition, Title: beat.Title,
			Color: beatColor, Description: beat.Description, Prompt: beat.Prompt, UpdatedAt: updatedAt,
		})

		var sources []*core.BeatRecapSource

		for scenePosition, scene := range beat.Scenes {
			addPromptScene(t, snapshot, scene, fixtureBeatID(beatPosition), scenePosition, position)

			source := &core.BeatRecapSource{SceneID: fixtureSceneID(position)}
			if lo.ContainsBy(scene.Cells, func(cell *promptCell) bool { return cell.Layer == 0 }) {
				source.CellUpdatedAt = updatedAt
			}

			sources = append(sources, source)
			position++
		}

		if beat.Recap != nil {
			recap, read := beat.Recap.recap()
			snapshot.BeatRecaps = append(snapshot.BeatRecaps, &core.BeatRecap{
				Recap: recap, BeatID: fixtureBeatID(beatPosition), BeatUpdatedAt: read, Sources: sources,
			})
		}
	}

	return fixture, snapshot
}

// addPromptScene adds a fixture scene, at a position in its beat, to the snapshot, with its cells and recaps.
// Narrative is the scene's position in narrative order, which names it.
func addPromptScene(
	t *testing.T, snapshot *core.ProjectSnapshot, scene *promptScene, beatID uuid.UUID, position, narrative int,
) {
	t.Helper()

	sceneID := fixtureSceneID(narrative)

	snapshot.Scenes = append(snapshot.Scenes, &core.Scene{
		ID: sceneID, ProjectID: projectID, BeatID: beatID, Position: position, Title: scene.Title,
		Description: scene.Description, Prompt: scene.Prompt, UpdatedAt: updatedAt,
	})

	for _, cell := range scene.Cells {
		content, err := json.Marshal(cell.Content)
		require.NoError(t, err)

		snapshot.Cells = append(snapshot.Cells, &core.Cell{
			ProjectID: projectID, SceneID: sceneID, LayerID: fixtureLayerID(cell.Layer), Content: content,
			UpdatedAt: updatedAt,
		})

		if cell.Recap != nil {
			recap, read := cell.Recap.recap()
			snapshot.Recaps = append(snapshot.Recaps, &core.CellRecap{
				Recap: recap, SceneID: sceneID, LayerID: fixtureLayerID(cell.Layer), CellUpdatedAt: read,
			})
		}
	}
}

// goldenVariants lists the requests the golden prompts render, each named after its generation kind.
var goldenVariants = []struct {
	name string
	kind string
}{
	{name: "suggest", kind: core.GenerationKindSuggest},
	{name: "consolidate", kind: core.GenerationKindConsolidate},
	{name: "consolidate-beat", kind: core.GenerationKindConsolidate},
	{name: "check", kind: core.GenerationKindCheck},
}

// promptRequest returns the request of a golden variant the fixture holds.
func (fixture *promptProject) promptRequest(
	variant, kind string, snapshot *core.ProjectSnapshot,
) *core.GenerationPlanRequest {
	request := &core.GenerationPlanRequest{Snapshot: snapshot, Kind: kind}

	switch variant {
	case core.GenerationKindSuggest:
		suggest := fixture.Requests.Suggest
		request.Instruction = suggest.Instruction

		for _, target := range suggest.Targets {
			planned := &core.GenerationTarget{
				SceneID: fixtureSceneID(target.Scene), Field: target.Field, Mode: target.Mode,
				Range: target.Range, Kept: target.Kept,
			}
			if target.Layer != nil {
				planned.LayerID = fixtureLayerID(*target.Layer)
			}

			request.Targets = append(request.Targets, planned)
		}

		for _, layer := range suggest.Exclude {
			request.Exclude = append(request.Exclude, fixtureLayerID(layer))
		}

		request.Overrides = &core.GenerationOverrides{}
		for _, scene := range suggest.Overrides.Scenes {
			request.Overrides.Scenes = append(request.Overrides.Scenes, &core.SceneOverride{
				ID: fixtureSceneID(scene.Scene), Title: scene.Title, Description: scene.Description,
			})
		}
	case "consolidate":
		cell := fixture.Requests.Consolidate.Cell
		request.Cell = &core.GenerationCell{SceneID: fixtureSceneID(cell.Scene), LayerID: fixtureLayerID(cell.Layer)}
	case "consolidate-beat":
		request.BeatID = fixtureBeatID(fixture.Requests.ConsolidateBeat.Beat)
	case core.GenerationKindCheck:
		request.Scenes = fixture.Requests.Check.Scenes
		request.Instruction = fixture.Requests.Check.Instruction
	}

	return request
}

// modelSuggestion is one suggestion of a suggest output, with the handles it names.
type modelSuggestion struct {
	Target string  `json:"target"`
	Text   *string `json:"text"`
	Edits  []struct {
		Block  *string         `json:"block"`
		Blocks json.RawMessage `json:"blocks"`
	} `json:"edits"`
}

// modelOutput is a model's output of any kind, with the handles it names.
type modelOutput struct {
	Manuscript []modelSuggestion `json:"manuscript"`
	Text       []modelSuggestion `json:"text"`
	Findings   []struct {
		Evidence []struct {
			Scene string `json:"scene"`
		} `json:"evidence"`
	} `json:"findings"`
}

// requireOutput fails unless output, the fixture output of a generation kind, passes the plan's strict
// schema and the full output schema, and unless every handle it names maps back through the plan to
// content that passes the content schemas. A suggest output answers each target once.
func requireOutput(t *testing.T, plan *core.ContextPlan, snapshot *core.ProjectSnapshot, output []byte) {
	t.Helper()

	strictSchema := &jsonschema.Schema{}
	require.NoError(t, json.Unmarshal(plan.Prompt.Schema, strictSchema))

	strict, err := strictSchema.Resolve(nil)
	require.NoError(t, err)

	var value any
	require.NoError(t, json.Unmarshal(output, &value))
	require.NoError(t, strict.Validate(value))

	full, err := schemas.Output(plan.Kind)
	require.NoError(t, err)
	require.NoError(t, full.Validate(output))

	parsed := &modelOutput{}
	require.NoError(t, json.Unmarshal(output, parsed))

	suggestions := slices.Concat(parsed.Manuscript, parsed.Text)
	if plan.Kind == core.GenerationKindSuggest {
		// The model answers each target once.
		require.ElementsMatch(t,
			lo.Map(plan.Targets, func(target *core.ContextTarget, _ int) string { return target.Handle }),
			lo.Map(suggestions, func(suggestion modelSuggestion, _ int) string { return suggestion.Target }))
	}

	for _, suggestion := range suggestions {
		target, ok := plan.Target(suggestion.Target)
		require.True(t, ok, suggestion.Target)

		for _, edit := range suggestion.Edits {
			if edit.Block != nil {
				_, ok = target.Block(*edit.Block)
				require.True(t, ok, *edit.Block)
			}

			// A delete carries no blocks.
			if len(edit.Blocks) > 0 {
				requireContent(t, core.LayerKindManuscript, `{"blocks":`+string(edit.Blocks)+`}`)
			}
		}

		if suggestion.Text != nil && target.LayerID != uuid.Nil {
			layer, ok := lo.Find(snapshot.Layers, func(layer *core.Layer) bool { return layer.ID == target.LayerID })
			require.True(t, ok)

			text, err := json.Marshal(map[string]string{"text": *suggestion.Text})
			require.NoError(t, err)
			requireContent(t, layer.Kind, string(text))
		}
	}

	for _, finding := range parsed.Findings {
		for _, quote := range finding.Evidence {
			scene, ok := plan.Scene(quote.Scene)
			require.True(t, ok, quote.Scene)
			require.Equal(t, core.ContextLevelFull, scene.Level, "a quote cites a full scene")
		}
	}
}

// requireContent fails unless content passes the content schema of a layer kind.
func requireContent(t *testing.T, kind, content string) {
	t.Helper()

	schema, err := schemas.Content(kind)
	require.NoError(t, err)
	require.NoError(t, schema.Validate(json.RawMessage(content)))
}

// goldenPrompt lays a prompt out as its golden file holds it.
func goldenPrompt(prompt *core.Prompt) string {
	call := fmt.Sprintf("model %s, reasoning %s, at most %d output tokens", prompt.Model, prompt.Reasoning,
		prompt.MaxOutputTokens)

	return "=== call\n" + call + "\n\n=== developer\n" + prompt.Developer + "\n\n=== schema\n" +
		string(prompt.Schema) + "\n\n=== frame\n" + prompt.Frame + "\n\n=== neighborhood\n" + prompt.Neighborhood +
		"\n\n=== task\n" + prompt.Task + "\n"
}

// sections joins what the prompt sends after its developer message.
func sections(prompt *core.Prompt) string {
	return prompt.Frame + "\n" + prompt.Neighborhood + "\n" + prompt.Task
}

// planService returns the planner of one generation kind with its settings.
func planService(kind string, settings config.GenerationKind) *core.GenerationPlan {
	return core.NewGenerationPlan(config.Generations{Kinds: map[string]config.GenerationKind{kind: settings}})
}

// longContextTokens is the input size past which OpenAI bills long context at twice the rate.
const longContextTokens = 272000

func TestGenerationPlan(t *testing.T) {
	t.Parallel()

	t.Run("Error/KindNotConfigured", func(t *testing.T) {
		t.Parallel()

		plan, err := core.NewGenerationPlan(config.Generations{}).Exec(t.Context(),
			suggestRequest(planProject(1), cellTarget(0, 0, core.TargetModeRevise)))
		require.Error(t, err)
		require.Nil(t, plan)
	})

	// A project at the store's scene limit, with a paragraph per scene, plans every kind within its default
	// budget: far beats collapse in the frame.
	t.Run("Success/LargeProject", func(t *testing.T) {
		t.Parallel()

		snapshot := planProject(2000)
		for index, scene := range snapshot.Scenes {
			scene.BeatID, scene.Position = fixtureBeatID(index/10), index%10
			scene.Description = strings.Repeat("A scene of the long story. ", 11)
		}

		snapshot.Beats = snapshot.Beats[:200]

		requests := []*core.GenerationPlanRequest{
			suggestRequest(snapshot, cellTarget(1000, 0, core.TargetModeRevise)),
			{Snapshot: snapshot, Kind: core.GenerationKindConsolidate, Cell: &core.GenerationCell{
				SceneID: fixtureSceneID(1000), LayerID: fixtureLayerID(0),
			}},
			{Snapshot: snapshot, Kind: core.GenerationKindCheck},
		}

		for _, request := range requests {
			settings := config.GenerationsConfigDefault.Kinds[request.Kind]

			plan, err := planService(request.Kind, settings).Exec(t.Context(), request)
			require.NoError(t, err, request.Kind)
			require.LessOrEqual(t, plan.Estimate, settings.Budget, request.Kind)
		}
	})

	for _, testCase := range generationPlanCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			settings := config.GenerationKind{Model: "a-model", Reasoning: "low", MaxOutputTokens: 1000, Limit: 1 << 30}
			if testCase.limit > 0 {
				settings.Limit = testCase.limit
			}

			fixed, err := planService(testCase.request.Kind, settings).Exec(t.Context(), testCase.request)
			if testCase.expectErr != nil {
				require.ErrorIs(t, err, testCase.expectErr)
				require.Nil(t, fixed)

				return
			}

			require.NoError(t, err)

			settings.Budget = lo.Ternary(testCase.unbounded, 1<<30, fixed.Estimate+testCase.extra)
			service := planService(testCase.request.Kind, settings)

			plan, err := service.Exec(t.Context(), testCase.request)
			require.NoError(t, err)
			require.LessOrEqual(t, plan.Estimate, plan.Budget)

			again, err := service.Exec(t.Context(), testCase.request)
			require.NoError(t, err)
			require.Equal(t, plan, again, "the same request gives the same plan")

			if testCase.expectLevels != nil {
				require.Equal(t, testCase.expectLevels, lo.Map(plan.Scenes, func(scene *core.ContextScene, _ int) string {
					return scene.Level
				}))
			}

			for _, text := range testCase.expectContains {
				require.Contains(t, sections(plan.Prompt), text)
			}

			for _, text := range testCase.expectAbsent {
				require.NotContains(t, sections(plan.Prompt), text)
			}

			if testCase.expect != nil {
				testCase.expect(t, plan)
			}
		})
	}

	for _, language := range core.ProjectLanguages {
		fixture, snapshot := loadPromptProject(t, language)

		for _, variant := range goldenVariants {
			kind := variant.kind

			t.Run("Success/Golden/"+variant.name+"."+language, func(t *testing.T) {
				t.Parallel()

				// The golden prompts run on the default settings of the kind, with a budget of their own.
				settings, ok := config.GenerationsConfigDefault.Kinds[kind]
				require.True(t, ok, "the kind has default settings")
				require.NotEmpty(t, settings.Model)
				require.Positive(t, settings.Budget)
				require.GreaterOrEqual(t, settings.Limit, settings.Budget)
				require.Less(t, settings.Limit, longContextTokens)

				settings.Budget = fixture.Budgets[variant.name]

				request := fixture.promptRequest(variant.name, kind, snapshot)

				plan, err := planService(kind, settings).Exec(t.Context(), request)
				require.NoError(t, err)

				name := variant.name + "." + language + ".golden"
				if *updateGolden {
					path := filepath.Join("testdata", "prompts", name)
					require.NoError(t, os.WriteFile(path, []byte(goldenPrompt(plan.Prompt)), 0o600))
				} else {
					golden, err := promptFixtures.ReadFile("testdata/prompts/" + name)
					require.NoError(t, err)
					require.Equal(t, string(golden), goldenPrompt(plan.Prompt))
				}

				require.LessOrEqual(t, plan.Estimate, plan.Budget)

				output, err := promptFixtures.ReadFile("testdata/prompts/" + kind + ".output.json")
				require.NoError(t, err)
				requireOutput(t, plan, snapshot, output)
			})
		}
	}
}
