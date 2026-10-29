package core

import (
	"fmt"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/config/prompts"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

// charsPerToken is how many characters [estimateTokens] counts as one token.
const charsPerToken = 4

// sceneRender is a scene rendered at one level, with its token estimate.
type sceneRender struct {
	text string
	cost int
}

// shownCell is a cell a scene shows at a level, whole or as its recap.
type shownCell struct {
	cell  *draftCell
	recap bool
}

// The prompt views are the data the templates of package prompts render. Handles stand for identifiers.
type (
	promptUser struct {
		Frame  string
		Scenes []string
		Task   string
	}

	promptFrame struct {
		Title    string
		Premise  string
		Language string
		Layers   []*promptLayer
		Scenes   []*promptFrameScene
	}

	promptLayer struct {
		Handle      string
		Kind        string
		Name        string
		Description string
	}

	// promptFrameScene is the line of a scene in the frame. A field the generation writes anew is empty.
	promptFrameScene struct {
		Handle      string
		Title       string
		Description string
	}

	promptScene struct {
		Handle string
		Level  string
		Prompt string
		Cells  []*promptCell
	}

	// promptCell is one cell of a neighborhood scene: its content, its recap, or the handle of the target
	// that shows it in the task.
	promptCell struct {
		Layer   string
		Target  string
		Recap   string
		Content *cellContent
	}

	promptTask struct {
		Targets     []*promptTarget
		Cell        *promptRecapCell
		Scenes      *promptSpan
		Instruction string
	}

	// promptTarget is one target of a suggest generation, with the content its mode shows.
	promptTarget struct {
		Handle string
		Scene  string
		Layer  string
		Field  string
		Mode   string

		HasText bool
		Text    string

		Manuscript bool
		Before     []*contentBlock
		Range      []*contentBlock
		After      []*contentBlock
	}

	promptRecapCell struct {
		Scene   string
		Layer   string
		Content *cellContent
	}

	promptSpan struct {
		First string
		Last  string
	}
)

// render gives each scene its level within budget, renders the prompt of the generation, and returns its
// plan.
func (draft *contextDraft) render(budget int) (*ContextPlan, error) {
	templates, err := prompts.Lookup(draft.request.Kind, draft.project.Language)
	if err != nil {
		return nil, fmt.Errorf("templates: %w", err)
	}

	output, err := schemas.Output(draft.request.Kind)
	if err != nil {
		return nil, fmt.Errorf("output schema: %w", err)
	}

	schema, err := output.Strict()
	if err != nil {
		return nil, fmt.Errorf("strict output schema: %w", err)
	}

	developer, err := execute(templates, "developer", nil)
	if err != nil {
		return nil, err
	}

	user, err := draft.renderUser(templates, budget-estimateTokens(developer)-estimateTokens(string(schema)))
	if err != nil {
		return nil, err
	}

	plan := draft.plan()
	plan.Budget = budget
	plan.Prompt = &Prompt{Developer: developer, Schema: schema, User: user}
	plan.Estimate = estimateTokens(developer) + estimateTokens(string(schema)) + estimateTokens(user)

	return plan, nil
}

// renderUser renders the user message within budget. The frame, the task and the scenes of targets
// always enter, and the other scenes share what they leave.
func (draft *contextDraft) renderUser(templates *template.Template, budget int) (string, error) {
	frame, err := execute(templates, "frame", draft.frameView())
	if err != nil {
		return "", err
	}

	task, err := execute(templates, "task."+draft.request.Kind, draft.taskView())
	if err != nil {
		return "", err
	}

	err = draft.renderScenes(templates)
	if err != nil {
		return "", err
	}

	budget -= estimateTokens(frame) + estimateTokens(task)

	for _, scene := range draft.scenes {
		if scene.pinned {
			scene.level = ContextLevelFull
			budget -= scene.renders[ContextLevelFull].cost
		}
	}

	allocate(draft.ranked, budget)

	var scenes []string

	for _, scene := range draft.scenes {
		if scene.level != ContextLevelOmitted {
			scenes = append(scenes, scene.renders[scene.level].text)
		}
	}

	return execute(templates, "user", &promptUser{Frame: frame, Scenes: scenes, Task: task})
}

// renderScenes renders each scene at every level it can enter at. A scene of a target enters whole, and
// a scene enters as a recap only when its manuscript has a fresh one.
func (draft *contextDraft) renderScenes(templates *template.Template) error {
	for _, scene := range draft.scenes {
		levels := []string{ContextLevelFull}

		if !scene.pinned {
			levels = append(levels, ContextLevelOutline)

			if scene.hasRecap() {
				levels = append(levels, ContextLevelRecap)
			}
		}

		scene.renders = make(map[string]*sceneRender, len(levels))

		for _, level := range levels {
			text, err := execute(templates, "scene", scene.view(level))
			if err != nil {
				return err
			}

			scene.renders[level] = &sceneRender{text: text, cost: estimateTokens(text)}
		}
	}

	return nil
}

// allocate gives each ranked scene the richest level that fits in budget. Most relevant first, every
// scene takes its outline while the budget allows; then, in the same order, each takes the richest level
// that still fits. A scene too large for what remains is passed over, and a smaller one after it may still
// fit, so a block is never cut.
func allocate(ranked []*draftScene, budget int) {
	for _, scene := range ranked {
		cost := scene.renders[ContextLevelOutline].cost
		if cost <= budget {
			scene.level = ContextLevelOutline
			budget -= cost
		}
	}

	for _, scene := range ranked {
		if scene.level != ContextLevelOutline {
			continue
		}

		for _, level := range []string{ContextLevelFull, ContextLevelRecap} {
			render, ok := scene.renders[level]
			if !ok {
				continue
			}

			extra := render.cost - scene.renders[ContextLevelOutline].cost
			if extra <= budget {
				scene.level = level
				budget -= extra

				break
			}
		}
	}
}

// hasRecap reports whether the scene can enter as a recap: one of its manuscript cells has a fresh recap.
func (scene *draftScene) hasRecap() bool {
	return lo.ContainsBy(scene.cells, func(cell *draftCell) bool {
		return cell.layer.layer.Kind == LayerKindManuscript && cell.recap != nil
	})
}

// shows lists the cells the scene shows at a level. A full scene shows every cell, a target's own cell as
// its handle. A recap scene shows its manuscript as fresh recaps, and an outline scene leaves it out.
func (scene *draftScene) shows(level string) []*shownCell {
	var shown []*shownCell

	for _, cell := range scene.cells {
		manuscript := cell.layer.layer.Kind == LayerKindManuscript

		switch {
		case level == ContextLevelFull, !manuscript:
			shown = append(shown, &shownCell{cell: cell})
		case level == ContextLevelRecap && cell.recap != nil:
			shown = append(shown, &shownCell{cell: cell, recap: true})
		}
	}

	return shown
}

// view returns what the template renders for the scene at a level. The scene of a target shows the
// author's prompt for it.
func (scene *draftScene) view(level string) *promptScene {
	view := &promptScene{Handle: scene.handle, Level: level}
	if scene.pinned {
		view.Prompt = scene.scene.Prompt
	}

	for _, shown := range scene.shows(level) {
		cell := &promptCell{Layer: shown.cell.layer.handle}

		switch {
		case shown.cell.target != nil:
			cell.Target = shown.cell.target.handle
		case shown.recap:
			cell.Recap = shown.cell.recap.Summary
		default:
			cell.Content = shown.cell.content
		}

		view.Cells = append(view.Cells, cell)
	}

	return view
}

// frameView returns what the template renders for the frame: the project, the layers the context shows,
// and every scene. A scene field the generation writes anew is hidden.
func (draft *contextDraft) frameView() *promptFrame {
	view := &promptFrame{Title: draft.project.Title, Premise: draft.project.Premise, Language: draft.project.Language}

	for _, layer := range draft.layers {
		if layer.excluded && !layer.target {
			continue
		}

		view.Layers = append(view.Layers, &promptLayer{
			Handle: layer.handle, Kind: layer.layer.Kind, Name: layer.layer.Name, Description: layer.layer.Description,
		})
	}

	lines := make(map[*draftScene]*promptFrameScene, len(draft.scenes))

	for _, scene := range draft.scenes {
		lines[scene] = &promptFrameScene{
			Handle: scene.handle, Title: scene.scene.Title, Description: scene.scene.Description,
		}
		view.Scenes = append(view.Scenes, lines[scene])
	}

	for _, target := range draft.targets {
		if target.mode != TargetModeWrite {
			continue
		}

		switch target.field {
		case SceneFieldTitle:
			lines[target.scene].Title = ""
		case SceneFieldDescription:
			lines[target.scene].Description = ""
		}
	}

	return view
}

// taskView returns what the template renders for the task of the generation's kind.
func (draft *contextDraft) taskView() *promptTask {
	view := &promptTask{Instruction: draft.request.Instruction}

	switch draft.request.Kind {
	case GenerationKindSuggest:
		view.Targets = lo.Map(draft.targets, func(target *draftTarget, _ int) *promptTarget { return target.view() })
	case GenerationKindConsolidate:
		target := draft.targets[0]
		view.Cell = &promptRecapCell{
			Scene: target.scene.handle, Layer: target.cell.layer.handle, Content: target.cell.content,
		}
	case GenerationKindCheck:
		if span := draft.request.Scenes; span != nil {
			view.Scenes = &promptSpan{First: draft.scenes[span.Start].handle, Last: draft.scenes[span.End-1].handle}
		}
	}

	return view
}

// view returns what the template renders for a suggest target. A text target shows its current text in
// revise mode. A manuscript target shows the blocks around its range, and the blocks of its range its
// mode shows.
func (target *draftTarget) view() *promptTarget {
	view := &promptTarget{Handle: target.handle, Scene: target.scene.handle, Field: target.field, Mode: target.mode}

	switch {
	case target.cell == nil:
		view.HasText = target.mode == TargetModeRevise
		view.Text = lo.Ternary(target.field == SceneFieldTitle, target.scene.scene.Title, target.scene.scene.Description)
	case target.span == nil:
		view.Layer = target.cell.layer.handle
		view.HasText = target.mode == TargetModeRevise
		view.Text = target.cell.content.Text
	default:
		view.Layer = target.cell.layer.handle
		view.Manuscript = true
		view.Before, view.Range, view.After = target.shown()
	}

	return view
}

// plan returns the plan of the context the draft renders.
func (draft *contextDraft) plan() *ContextPlan {
	plan := &ContextPlan{Kind: draft.request.Kind}

	for _, scene := range draft.scenes {
		planned := &ContextScene{
			Handle: scene.handle, SceneID: scene.scene.ID, Level: scene.level, Override: scene.override,
			UpdatedAt: lo.Ternary(scene.override, time.Time{}, scene.scene.UpdatedAt),
		}

		if scene.level != ContextLevelOmitted {
			for _, shown := range scene.shows(scene.level) {
				if shown.cell.target != nil {
					continue
				}

				planned.Cells = append(planned.Cells, &ContextCell{
					LayerID: shown.cell.layer.layer.ID, Recap: shown.recap,
					Override: shown.cell.override, UpdatedAt: shown.cell.updatedAt,
				})
			}
		}

		plan.Scenes = append(plan.Scenes, planned)
	}

	for _, layer := range draft.layers {
		if !layer.excluded || layer.target {
			plan.Layers = append(plan.Layers, &ContextLayer{Handle: layer.handle, LayerID: layer.layer.ID})
		}
	}

	plan.Targets = lo.Map(draft.targets, func(target *draftTarget, _ int) *ContextTarget { return target.plan() })

	return plan
}

// plan returns the plan of the target.
func (target *draftTarget) plan() *ContextTarget {
	planned := &ContextTarget{
		Handle: target.handle, SceneID: target.scene.scene.ID, Field: target.field, Mode: target.mode,
		Range: target.span, Kept: target.kept, Blocks: target.blocks,
	}

	if target.cell == nil {
		planned.Override = target.scene.override
		planned.UpdatedAt = lo.Ternary(target.scene.override, time.Time{}, target.scene.scene.UpdatedAt)

		return planned
	}

	planned.LayerID = target.cell.layer.layer.ID
	planned.Override, planned.UpdatedAt = target.cell.override, target.cell.updatedAt

	return planned
}

// execute renders the template name of templates with data.
func execute(templates *template.Template, name string, data any) (string, error) {
	var builder strings.Builder

	err := templates.ExecuteTemplate(&builder, name, data)
	if err != nil {
		return "", fmt.Errorf("execute %s: %w", name, err)
	}

	return builder.String(), nil
}

// estimateTokens estimates how many tokens a model reads in text, from its length in characters.
func estimateTokens(text string) int {
	return (utf8.RuneCountInString(text) + charsPerToken - 1) / charsPerToken
}
