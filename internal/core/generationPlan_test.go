package core_test

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	Scenes []struct {
		Title       string `yaml:"title"`
		Description string `yaml:"description"`
		Prompt      string `yaml:"prompt"`
		Cells       []struct {
			Layer   int `yaml:"layer"`
			Content any `yaml:"content"`
			Recap   *struct {
				Summary string   `yaml:"summary"`
				Names   []string `yaml:"names"`
				Stale   bool     `yaml:"stale"`
			} `yaml:"recap"`
		} `yaml:"cells"`
	} `yaml:"scenes"`
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
		Check struct {
			Scenes      *core.Range `yaml:"scenes"`
			Instruction string      `yaml:"instruction"`
		} `yaml:"check"`
	} `yaml:"requests"`
}

// fixtureSceneID and fixtureLayerID give the scene and the layer at a position of a fixture their ID.
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

	for position, scene := range fixture.Scenes {
		snapshot.Scenes = append(snapshot.Scenes, &core.Scene{
			ID: fixtureSceneID(position), ProjectID: projectID, Position: position, Title: scene.Title,
			Color: sceneColor, Description: scene.Description, Prompt: scene.Prompt, UpdatedAt: updatedAt,
		})

		for _, cell := range scene.Cells {
			content, err := json.Marshal(cell.Content)
			require.NoError(t, err)

			snapshot.Cells = append(snapshot.Cells, &core.Cell{
				ProjectID: projectID, SceneID: fixtureSceneID(position), LayerID: fixtureLayerID(cell.Layer),
				Content: content, UpdatedAt: updatedAt,
			})

			if cell.Recap != nil {
				read := updatedAt
				if cell.Recap.Stale {
					read = updatedAt.Add(-time.Hour)
				}

				snapshot.Recaps = append(snapshot.Recaps, &core.CellRecap{
					SceneID: fixtureSceneID(position), LayerID: fixtureLayerID(cell.Layer),
					Summary: cell.Recap.Summary, Names: cell.Recap.Names, ContentUpdatedAt: read,
				})
			}
		}
	}

	return fixture, snapshot
}

// promptRequest returns the request of a generation kind the fixture holds.
func (fixture *promptProject) promptRequest(kind string, snapshot *core.ProjectSnapshot) *core.GenerationPlanRequest {
	request := &core.GenerationPlanRequest{Snapshot: snapshot, Kind: kind}

	switch kind {
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
	case core.GenerationKindConsolidate:
		cell := fixture.Requests.Consolidate.Cell
		request.Cell = &core.GenerationCell{SceneID: fixtureSceneID(cell.Scene), LayerID: fixtureLayerID(cell.Layer)}
	case core.GenerationKindCheck:
		request.Scenes = fixture.Requests.Check.Scenes
		request.Instruction = fixture.Requests.Check.Instruction
	}

	return request
}

// modelOutput is a model's output of any kind, with the handles it names.
type modelOutput struct {
	Suggestions []struct {
		Target string  `json:"target"`
		Text   *string `json:"text"`
		Edits  []struct {
			Block  *string         `json:"block"`
			Blocks json.RawMessage `json:"blocks"`
		} `json:"edits"`
	} `json:"suggestions"`
	Findings []struct {
		Evidence []struct {
			Scene string `json:"scene"`
		} `json:"evidence"`
	} `json:"findings"`
}

// requireOutput fails unless output, the fixture output of a generation kind, passes the plan's strict
// schema and the full output schema, and unless every handle it names maps back through the plan to
// content that passes the content schemas.
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

	for _, suggestion := range parsed.Suggestions {
		target, ok := plan.Target(suggestion.Target)
		require.True(t, ok, suggestion.Target)

		for _, edit := range suggestion.Edits {
			if edit.Block != nil {
				_, ok = target.Block(*edit.Block)
				require.True(t, ok, *edit.Block)
			}

			requireContent(t, core.LayerKindManuscript, `{"blocks":`+string(edit.Blocks)+`}`)
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
	return "=== developer\n" + prompt.Developer + "\n\n=== schema\n" + string(prompt.Schema) +
		"\n\n=== user\n" + prompt.User + "\n"
}

func TestGenerationPlan(t *testing.T) {
	t.Parallel()

	for _, testCase := range generationPlanCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := core.NewGenerationPlan(config.Generations{Budgets: map[string]int{testCase.request.Kind: 0}})

			fixed, err := service.Exec(t.Context(), testCase.request)
			if testCase.expectErr != nil {
				require.ErrorIs(t, err, testCase.expectErr)
				require.Nil(t, fixed)

				return
			}

			require.NoError(t, err)

			budget := lo.Ternary(testCase.unbounded, 1<<30, fixed.Estimate+testCase.extra)
			service = core.NewGenerationPlan(config.Generations{Budgets: map[string]int{testCase.request.Kind: budget}})

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
				require.Contains(t, plan.Prompt.User, text)
			}

			for _, text := range testCase.expectAbsent {
				require.NotContains(t, plan.Prompt.User, text)
			}

			if testCase.expect != nil {
				testCase.expect(t, plan)
			}
		})
	}

	for _, language := range core.ProjectLanguages {
		fixture, snapshot := loadPromptProject(t, language)

		service := core.NewGenerationPlan(config.Generations{Budgets: fixture.Budgets})

		for _, kind := range core.GenerationKinds {
			t.Run("Success/Golden/"+kind+"."+language, func(t *testing.T) {
				t.Parallel()

				require.Positive(t, config.GenerationsConfigDefault.Budgets[kind], "the default budget of the kind")

				plan, err := service.Exec(t.Context(), fixture.promptRequest(kind, snapshot))
				require.NoError(t, err)

				name := kind + "." + language + ".golden"
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
