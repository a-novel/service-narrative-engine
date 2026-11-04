package core_test

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
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

// storedProject is a project as a planner test stores it: what the snapshot of a generation holds but the
// cell summaries, and every cell whole, with its recap. [planOf] summarizes the cells.
type storedProject struct {
	*core.ProjectSnapshot

	cells []*core.Cell
}

// recap gives the cell of a scene in a layer a recap, which read the cell at an instant.
func (project *storedProject) recap(sceneID, layerID uuid.UUID, recap core.Recap, read time.Time) {
	cell, ok := lo.Find(project.cells, func(cell *core.Cell) bool {
		return cell.SceneID == sceneID && cell.LayerID == layerID
	})
	if !ok {
		panic("recap of a cell the fixture does not hold")
	}

	cell.Recap, cell.RecapCellUpdatedAt = &recap, read
}

// planRequest is a request of [TestGenerationPlan], with the cells of its project, which the planner reads
// through a fake of its DAO.
type planRequest struct {
	*core.GenerationPlanRequest

	cells []*core.Cell
}

// planOf returns a request on a stored project, whose snapshot summarizes the project's cells as
// [dao.PgCellSummaryList] does.
func planOf(project *storedProject, request *core.GenerationPlanRequest) *planRequest {
	snapshot := *project.ProjectSnapshot
	snapshot.Cells = lo.Map(project.cells, func(cell *core.Cell, _ int) *core.CellSummary { return summarize(cell) })
	request.Snapshot = &snapshot

	return &planRequest{GenerationPlanRequest: request, cells: project.cells}
}

// summarize returns the summary of a stored cell, as [dao.PgCellSummaryList] measures it: every text,
// speaker and tone counts, and every speaker appears once, in order.
func summarize(cell *core.Cell) *core.CellSummary {
	var content struct {
		Text   string `json:"text"`
		Blocks []struct {
			Text    string `json:"text"`
			Speaker string `json:"speaker"`
			Parts   []struct {
				Text string `json:"text"`
				Tone string `json:"tone"`
			} `json:"parts"`
		} `json:"blocks"`
	}

	err := json.Unmarshal(cell.Content, &content)
	if err != nil {
		panic(err)
	}

	summary := &core.CellSummary{
		SceneID: cell.SceneID, LayerID: cell.LayerID, UpdatedAt: cell.UpdatedAt,
		Empty: content.Text == "" && len(content.Blocks) == 0, Size: utf8.RuneCountInString(content.Text),
		Speakers: []string{},
	}

	for _, block := range content.Blocks {
		summary.Size += utf8.RuneCountInString(block.Text) + utf8.RuneCountInString(block.Speaker)

		for _, part := range block.Parts {
			summary.Size += utf8.RuneCountInString(part.Text) + utf8.RuneCountInString(part.Tone)
		}

		if block.Speaker != "" && !slices.Contains(summary.Speakers, block.Speaker) {
			summary.Speakers = append(summary.Speakers, block.Speaker)
		}
	}

	slices.Sort(summary.Speakers)

	if recap := cell.Recap; recap != nil {
		summary.Recap = &core.RecapSummary{
			Names: recap.Names, CondensedSize: utf8.RuneCountInString(recap.Condensed),
			CompactSize: utf8.RuneCountInString(recap.Compact), EssentialsSize: utf8.RuneCountInString(recap.Essentials),
			CellUpdatedAt: cell.RecapCellUpdatedAt,
		}
	}

	return summary
}

// storedKey names a stored cell by its scene and its layer.
type storedKey struct {
	sceneID uuid.UUID
	layerID uuid.UUID
}

// cellReads records what a planner read through its DAO: the cells whose content it read, the recap
// versions it read, and how many queries it ran.
type cellReads struct {
	mu       sync.Mutex
	contents []storedKey
	recaps   []string
	queries  int
}

// plan plans the request on the settings of its kind. The planner reads the request's cells through a fake
// of its DAO, which records the reads.
func (request *planRequest) plan(t *testing.T, settings config.GenerationKind) (*core.ContextPlan, *cellReads, error) {
	t.Helper()

	stored := lo.KeyBy(request.cells, func(cell *core.Cell) storedKey { return storedKey{cell.SceneID, cell.LayerID} })
	reads := &cellReads{}

	daoCells := coremocks.NewMockGenerationPlanDaoCells(t)
	daoCells.EXPECT().
		Exec(mock.Anything, mock.MatchedBy(func(read *dao.CellsSelectRequest) bool { return read.ProjectID == projectID })).
		RunAndReturn(func(_ context.Context, read *dao.CellsSelectRequest) ([]*dao.CellsSelectRow, error) {
			return reads.serve(stored, read), nil
		}).
		Maybe()

	service := core.NewGenerationPlan(
		config.Generations{Kinds: map[string]config.GenerationKind{request.Kind: settings}}, daoCells,
	)

	plan, err := service.Exec(t.Context(), request.GenerationPlanRequest)

	return plan, reads, err
}

// serve answers a read of stored cells as [dao.PgCellsSelect] does, and records it.
func (reads *cellReads) serve(stored map[storedKey]*core.Cell, read *dao.CellsSelectRequest) []*dao.CellsSelectRow {
	reads.mu.Lock()
	defer reads.mu.Unlock()

	reads.queries++

	var rows []*dao.CellsSelectRow

	for _, key := range read.Cells {
		cell, ok := stored[storedKey{key.SceneID, key.LayerID}]
		if !ok {
			continue
		}

		row := &dao.CellsSelectRow{SceneID: cell.SceneID, LayerID: cell.LayerID}

		if key.Content {
			row.Content = cell.Content
			reads.contents = append(reads.contents, storedKey{cell.SceneID, cell.LayerID})
		}

		if key.Recap != "" {
			row.RecapVersion = key.Recap
			reads.recaps = append(reads.recaps, key.Recap)

			if cell.Recap != nil {
				version := map[string]string{
					core.ContextLevelCondensed: cell.Recap.Condensed, core.ContextLevelCompact: cell.Recap.Compact,
					core.ContextLevelEssentials: cell.Recap.Essentials,
				}[key.Recap]
				row.Recap = &version
			}
		}

		rows = append(rows, row)
	}

	return rows
}

// loadPromptProject reads the fixture of a language, and returns its project and budgets.
func loadPromptProject(t *testing.T, language string) (*promptProject, *storedProject) {
	t.Helper()

	data, err := promptFixtures.ReadFile("testdata/prompts/project." + language + ".yaml")
	require.NoError(t, err)

	fixture := &promptProject{}
	require.NoError(t, yaml.Unmarshal(data, fixture))

	snapshot := &storedProject{ProjectSnapshot: &core.ProjectSnapshot{Project: &core.Project{
		ID: projectID, OwnerID: ownerID, Title: fixture.Project.Title, Premise: fixture.Project.Premise,
		Language: fixture.Project.Language, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}}}

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
	t *testing.T, snapshot *storedProject, scene *promptScene, beatID uuid.UUID, position, narrative int,
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

		snapshot.cells = append(snapshot.cells, &core.Cell{
			ProjectID: projectID, SceneID: sceneID, LayerID: fixtureLayerID(cell.Layer), Content: content,
			UpdatedAt: updatedAt,
		})

		if cell.Recap != nil {
			recap, read := cell.Recap.recap()
			snapshot.recap(sceneID, fixtureLayerID(cell.Layer), recap, read)
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
func (fixture *promptProject) promptRequest(variant, kind string, project *storedProject) *planRequest {
	request := planOf(project, &core.GenerationPlanRequest{Kind: kind})

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

// requireOutput fails unless output, the output of one call of a generation kind, passes the call's strict
// schema and the full output schema, and unless every handle it names maps back through the call to
// content that passes the content schemas. A suggest output answers each target once.
func requireOutput(t *testing.T, kind string, call *core.ContextCall, snapshot *core.ProjectSnapshot, output []byte) {
	t.Helper()

	strictSchema := &jsonschema.Schema{}
	require.NoError(t, json.Unmarshal(call.Prompt.Schema, strictSchema))

	strict, err := strictSchema.Resolve(nil)
	require.NoError(t, err)

	var value any
	require.NoError(t, json.Unmarshal(output, &value))
	require.NoError(t, strict.Validate(value))

	full, err := schemas.Output(kind)
	require.NoError(t, err)
	require.NoError(t, full.Validate(output))

	parsed := &modelOutput{}
	require.NoError(t, json.Unmarshal(output, parsed))

	suggestions := slices.Concat(parsed.Manuscript, parsed.Text)
	if kind == core.GenerationKindSuggest {
		// The model answers each target once.
		require.ElementsMatch(t,
			lo.Map(call.Targets, func(target *core.ContextTarget, _ int) string { return target.Handle }),
			lo.Map(suggestions, func(suggestion modelSuggestion, _ int) string { return suggestion.Target }))
	}

	for _, suggestion := range suggestions {
		target, ok := call.Target(suggestion.Target)
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
			scene, ok := call.Scene(quote.Scene)
			require.True(t, ok, quote.Scene)
			require.Equal(t, core.ContextLevelFull, scene.Level, "a quote cites a full scene")
		}
	}
}

// requireHandles fails unless every handle keeps one meaning across the calls of a plan: a scene or a target
// names the same thing in each call, and a block takes its handle in one call only.
func requireHandles(t *testing.T, plan *core.ContextPlan) {
	t.Helper()

	scenes := map[string]uuid.UUID{}
	targets := map[string]string{}
	blocks := map[string]bool{}

	for _, call := range plan.Calls {
		for _, scene := range call.Scenes {
			if id, ok := scenes[scene.Handle]; ok {
				require.Equal(t, id, scene.SceneID, scene.Handle)
			}

			scenes[scene.Handle] = scene.SceneID
		}

		for _, target := range call.Targets {
			named := target.SceneID.String() + target.LayerID.String() + target.BeatID.String() + target.Field
			if other, ok := targets[target.Handle]; ok {
				require.Equal(t, other, named, target.Handle)
			}

			targets[target.Handle] = named

			for _, block := range target.Blocks {
				require.False(t, blocks[block.Handle], block.Handle)

				blocks[block.Handle] = true
			}
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

// planGoldensPath holds the SHA-256 of every plan [TestGenerationPlan] makes, one line per subtest.
const planGoldensPath = "testdata/plans.golden"

// planGoldens pins each plan of [TestGenerationPlan] whole, its contexts and its prompts, by the SHA-256 of
// its JSON. With -update, the test records the plans it makes instead.
type planGoldens struct {
	mu     sync.Mutex
	hashes map[string]string
}

// loadPlanGoldens reads [planGoldensPath]. With -update, it rewrites the file once the test ends, keeping the
// lines of the subtests a filter skipped.
func loadPlanGoldens(t *testing.T) *planGoldens {
	t.Helper()

	goldens := &planGoldens{hashes: map[string]string{}}

	data, err := os.ReadFile(planGoldensPath)
	require.NoError(t, err)

	for line := range strings.Lines(string(data)) {
		hash, name, ok := strings.Cut(strings.TrimSpace(line), "  ")
		require.True(t, ok, line)

		goldens.hashes[name] = hash
	}

	if *updateGolden {
		t.Cleanup(func() {
			names := slices.Sorted(maps.Keys(goldens.hashes))

			var builder strings.Builder

			for _, name := range names {
				builder.WriteString(goldens.hashes[name] + "  " + name + "\n")
			}

			require.NoError(t, os.WriteFile(planGoldensPath, []byte(builder.String()), 0o600))
		})
	}

	return goldens
}

// check fails unless the plan a subtest made matches its golden, or records it with -update.
func (goldens *planGoldens) check(t *testing.T, name string, plan *core.ContextPlan) {
	t.Helper()

	data, err := json.Marshal(plan) //nolint:musttag // a test fingerprint, not a wire format
	require.NoError(t, err)

	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])

	goldens.mu.Lock()
	defer goldens.mu.Unlock()

	if *updateGolden {
		goldens.hashes[name] = hash

		return
	}

	require.Equal(t, goldens.hashes[name], hash, "the plan of %s changed", name)
}

// longContextTokens is the input size past which OpenAI bills long context at twice the rate.
const longContextTokens = 272000

func TestGenerationPlan(t *testing.T) {
	t.Parallel()

	goldens := loadPlanGoldens(t)

	t.Run("Error/KindNotConfigured", func(t *testing.T) {
		t.Parallel()

		plan, err := core.NewGenerationPlan(config.Generations{}, coremocks.NewMockGenerationPlanDaoCells(t)).
			Exec(t.Context(), suggestRequest(planProject(1), cellTarget(0, 0, core.TargetModeRevise)).GenerationPlanRequest)
		require.Error(t, err)
		require.Nil(t, plan)
	})

	t.Run("Error/ReadCells", func(t *testing.T) {
		t.Parallel()

		errFoo := errors.New("foo")

		daoCells := coremocks.NewMockGenerationPlanDaoCells(t)
		daoCells.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil, errFoo)

		request := suggestRequest(planProject(2), cellTarget(0, 0, core.TargetModeRevise))
		service := core.NewGenerationPlan(
			config.Generations{Kinds: map[string]config.GenerationKind{request.Kind: splitSettings()}}, daoCells,
		)

		plan, err := service.Exec(t.Context(), request.GenerationPlanRequest)
		require.ErrorIs(t, err, errFoo)
		require.Nil(t, plan)
	})

	// A cell deleted after the snapshot reads as empty, as a read outside a transaction may see a write.
	t.Run("Success/CellGone", func(t *testing.T) {
		t.Parallel()

		request := suggestRequest(planProject(2), cellTarget(0, 0, core.TargetModeRevise))
		request.cells = lo.Reject(request.cells, func(cell *core.Cell, _ int) bool {
			return cell.SceneID == fixtureSceneID(1) && cell.LayerID == fixtureLayerID(0)
		})

		settings := splitSettings()
		settings.Budget = 1 << 30

		plan, _, err := request.plan(t, settings)
		require.NoError(t, err)

		gone := `<scene handle="S2" level="full">` + "\n" + `<cell layer="L1"></cell>`
		require.Contains(t, plan.Calls[0].Prompt.Neighborhood, gone)
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

		requests := []*planRequest{
			suggestRequest(snapshot, cellTarget(1000, 0, core.TargetModeRevise)),
			planOf(snapshot, &core.GenerationPlanRequest{Kind: core.GenerationKindConsolidate, Cell: &core.GenerationCell{
				SceneID: fixtureSceneID(1000), LayerID: fixtureLayerID(0),
			}}),
			planOf(snapshot, &core.GenerationPlanRequest{Kind: core.GenerationKindCheck}),
		}

		for index, request := range requests {
			settings := config.GenerationsConfigDefault.Kinds[request.Kind]

			plan, reads, err := request.plan(t, settings)
			require.NoError(t, err, request.Kind)
			goldens.check(t, t.Name()+"/"+strconv.Itoa(index), plan)

			for _, call := range plan.Calls {
				require.LessOrEqual(t, call.Estimate, settings.Budget, request.Kind)
			}

			// A plan reads a few of the 4,000 cells in a few queries, but a check of every scene, which
			// shows each scene whole in one of its calls.
			require.LessOrEqual(t, reads.queries, 10, request.Kind)

			if request.Kind == core.GenerationKindCheck {
				require.Len(t, lo.Uniq(reads.contents), len(request.cells))
			} else {
				require.Less(t, len(reads.contents), len(request.cells)/10, request.Kind)
			}
		}
	})

	// A project at the store's limits plans every kind on the default settings, which turn on long context
	// and splitting: 200 beats of 10 scenes, twenty targets of 200,000 characters that pass the long limit
	// together, a scene at the 400,000-character cap, and a beat whose scenes carry the largest recaps.
	t.Run("Success/StoreLimits", func(t *testing.T) {
		t.Parallel()

		snapshot := planProject(2000)
		for index, scene := range snapshot.Scenes {
			scene.BeatID, scene.Position = fixtureBeatID(index/10), index%10
		}

		snapshot.Beats = snapshot.Beats[:200]

		large := blocks(lo.Times(100, func(int) string { return narration(strings.Repeat("Wind. ", 333)) })...)
		targets := lo.Times(20, func(index int) *core.GenerationTarget {
			setManuscript(snapshot, index*100, large)

			return cellTarget(index*100, 0, core.TargetModeRevise)
		})

		setManuscript(snapshot, 1005, blocks(lo.Times(200, func(int) string {
			return narration(strings.Repeat("Wind. ", 333))
		})...))

		for scene := 1010; scene < 1020; scene++ {
			snapshot.recap(fixtureSceneID(scene), fixtureLayerID(0), core.Recap{
				Condensed: "Wind.", Compact: strings.Repeat("Wind. ", 6666), Essentials: "Wind.",
			}, updatedAt)
		}

		requests := []*planRequest{
			suggestRequest(snapshot, targets...),
			planOf(snapshot, &core.GenerationPlanRequest{Kind: core.GenerationKindConsolidate, Cell: &core.GenerationCell{
				SceneID: fixtureSceneID(1005), LayerID: fixtureLayerID(0),
			}}),
			planOf(snapshot, &core.GenerationPlanRequest{Kind: core.GenerationKindConsolidate, BeatID: fixtureBeatID(101)}),
			planOf(snapshot, &core.GenerationPlanRequest{Kind: core.GenerationKindCheck}),
		}

		for index, request := range requests {
			settings := config.GenerationsConfigDefault.Kinds[request.Kind]

			plan, _, err := request.plan(t, settings)
			require.NoError(t, err, request.Kind)
			requireHandles(t, plan)
			goldens.check(t, t.Name()+"/"+strconv.Itoa(index), plan)

			for _, call := range plan.Calls {
				require.LessOrEqual(t, call.Estimate, settings.LongLimit, request.Kind)
			}
		}
	})

	// A story of many scenes, recaps and layers plans each kind on its default settings, within a budget that
	// leaves a mix of levels.
	for _, story := range storyRequests() {
		t.Run("Success/Story/"+story.name, func(t *testing.T) {
			t.Parallel()

			settings := config.GenerationsConfigDefault.Kinds[story.request.Kind]
			settings.Budget = story.budget

			plan, reads, err := story.request.plan(t, settings)
			require.NoError(t, err)
			requireHandles(t, plan)
			goldens.check(t, t.Name(), plan)

			for _, call := range plan.Calls {
				require.LessOrEqual(t, call.Estimate, plan.Budget)
			}

			// The allocator reads ahead, so the scenes past the edge of the budget share a few queries.
			require.LessOrEqual(t, reads.queries, 10)

			if story.request.Scenes == nil && story.request.Kind == core.GenerationKindCheck {
				return
			}

			// A plan that shows part of the story reads less than half of its content.
			stored := lo.KeyBy(story.request.cells, func(cell *core.Cell) storedKey {
				return storedKey{cell.SceneID, cell.LayerID}
			})
			read := lo.SumBy(lo.Uniq(reads.contents), func(key storedKey) int { return len(stored[key].Content) })
			total := lo.SumBy(story.request.cells, func(cell *core.Cell) int { return len(cell.Content) })
			require.Less(t, read, total/2)
		})
	}

	for _, testCase := range generationPlanSplitCases() {
		t.Run("Split/"+testCase.name, func(t *testing.T) {
			t.Parallel()

			single, _, err := testCase.request.plan(t, splitSettings())
			require.NoError(t, err)
			require.Len(t, single.Calls, 1)

			plan, _, err := testCase.request.plan(t, testCase.settings(single.Calls[0].Estimate))
			if testCase.expectErr != nil {
				require.ErrorIs(t, err, testCase.expectErr)
				require.Nil(t, plan)

				return
			}

			require.NoError(t, err)
			requireHandles(t, plan)
			goldens.check(t, t.Name(), plan)
			testCase.expect(t, plan)
		})
	}

	for _, testCase := range generationPlanCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			settings := config.GenerationKind{Model: "a-model", Reasoning: "low", MaxOutputTokens: 1 << 20, Limit: 1 << 30}
			if testCase.limit > 0 {
				settings.Limit = testCase.limit
			}

			fixed, _, err := testCase.request.plan(t, settings)
			if testCase.expectErr != nil {
				require.ErrorIs(t, err, testCase.expectErr)
				require.Nil(t, fixed)

				return
			}

			require.NoError(t, err)

			require.Len(t, fixed.Calls, 1)

			settings.Budget = lo.Ternary(testCase.unbounded, 1<<30, fixed.Calls[0].Estimate+testCase.extra)

			plan, _, err := testCase.request.plan(t, settings)
			require.NoError(t, err)
			require.Len(t, plan.Calls, 1)

			call := plan.Calls[0]
			require.LessOrEqual(t, call.Estimate, plan.Budget)

			again, _, err := testCase.request.plan(t, settings)
			require.NoError(t, err)
			require.Equal(t, plan, again, "the same request gives the same plan")
			goldens.check(t, t.Name(), plan)

			if testCase.expectLevels != nil {
				require.Equal(t, testCase.expectLevels, lo.Map(call.Scenes, func(scene *core.ContextScene, _ int) string {
					return scene.Level
				}))
			}

			for _, text := range testCase.expectContains {
				require.Contains(t, sections(call.Prompt), text)
			}

			for _, text := range testCase.expectAbsent {
				require.NotContains(t, sections(call.Prompt), text)
			}

			if testCase.expect != nil {
				testCase.expect(t, call)
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

				plan, _, err := request.plan(t, settings)
				require.NoError(t, err)
				require.Len(t, plan.Calls, 1)
				goldens.check(t, t.Name(), plan)

				call := plan.Calls[0]

				name := variant.name + "." + language + ".golden"
				if *updateGolden {
					path := filepath.Join("testdata", "prompts", name)
					require.NoError(t, os.WriteFile(path, []byte(goldenPrompt(call.Prompt)), 0o600))
				} else {
					golden, err := promptFixtures.ReadFile("testdata/prompts/" + name)
					require.NoError(t, err)
					require.Equal(t, string(golden), goldenPrompt(call.Prompt))
				}

				require.LessOrEqual(t, call.Estimate, plan.Budget)

				output, err := promptFixtures.ReadFile("testdata/prompts/" + kind + ".output.json")
				require.NoError(t, err)
				requireOutput(t, kind, call, snapshot.ProjectSnapshot, output)
			})
		}
	}
}
