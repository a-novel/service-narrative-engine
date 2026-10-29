package core

import (
	"fmt"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/config/prompts"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

// charsPerToken is how many characters [estimateTokens] counts as one token. The planner measures every
// part of a prompt in characters, which add up exactly, and estimates tokens once.
const charsPerToken = 4

// sceneRender is a scene rendered at one level. Size counts its characters with the line it takes in the
// neighborhood.
type sceneRender struct {
	text string
	size int
}

// shownCell is a cell a scene shows at a level, whole or as its recap.
type shownCell struct {
	cell  *draftCell
	recap bool
}

// contextSide is one side of a manuscript target's range, where the task shows context blocks.
type contextSide struct {
	target *draftTarget
	before bool
}

// The prompt views are the data the templates of package prompts render. Handles stand for identifiers.
type (
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

	// promptTarget is one target of a suggest generation, with the content its mode shows. The context of a
	// manuscript target holds the blocks nearest its range, and counts the ones it leaves out.
	promptTarget struct {
		Handle string
		Scene  string
		Layer  string
		Field  string
		Mode   string

		HasText bool
		Text    string

		Manuscript    bool
		HasBefore     bool
		Before        []*contentBlock
		BeforeOmitted int
		Range         []*contentBlock
		HasAfter      bool
		After         []*contentBlock
		AfterOmitted  int
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

// render plans what the generation reads within the budget of its kind, and renders its prompt.
//
// The frame, the task and the scenes of the targets always enter, unless they alone outgrow the kind's
// limit, which [ErrContextTooLarge] refuses. The blocks around each target's range come next, nearest first,
// and the other scenes share what is left.
func (draft *contextDraft) render(kind config.GenerationKind) (*ContextPlan, error) {
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

	prompt := &Prompt{
		Model: kind.Model, Reasoning: kind.Reasoning, MaxOutputTokens: kind.MaxOutputTokens, Schema: schema,
	}

	err = draft.renderSections(templates, prompt, kind)
	if err != nil {
		return nil, err
	}

	plan := draft.plan()
	plan.Budget = kind.Budget
	plan.Prompt = prompt
	plan.Estimate = estimateTokens(prompt.size())

	return plan, nil
}

// renderSections renders the developer message and the three sections of the prompt within the kind's
// budget.
func (draft *contextDraft) renderSections(
	templates *template.Template, prompt *Prompt, kind config.GenerationKind,
) error {
	var err error

	prompt.Developer, err = execute(templates, "developer", nil)
	if err != nil {
		return err
	}

	prompt.Frame, err = execute(templates, "frame", draft.frameView())
	if err != nil {
		return err
	}

	err = draft.renderPinned(templates)
	if err != nil {
		return err
	}

	// The task shows no context yet, and the neighborhood only the scenes of the targets.
	prompt.Task, err = execute(templates, "task."+draft.request.Kind, draft.taskView())
	if err != nil {
		return err
	}

	prompt.Neighborhood, err = draft.neighborhood(templates)
	if err != nil {
		return err
	}

	if prompt.size() > kind.Limit*charsPerToken {
		return fmt.Errorf(
			"%w: %d tokens, over the limit of %d", ErrContextTooLarge, estimateTokens(prompt.size()), kind.Limit,
		)
	}

	err = draft.showTargetContext(templates, kind.Budget*charsPerToken-prompt.size())
	if err != nil {
		return err
	}

	// The context only adds its blocks to the task, so the prompt stays within what it measured.
	prompt.Task, err = execute(templates, "task."+draft.request.Kind, draft.taskView())
	if err != nil {
		return err
	}

	err = draft.allocate(templates, kind.Budget*charsPerToken-prompt.size())
	if err != nil {
		return err
	}

	prompt.Neighborhood, err = draft.neighborhood(templates)

	return err
}

// renderPinned renders the scenes of the targets, which always enter whole.
func (draft *contextDraft) renderPinned(templates *template.Template) error {
	for _, scene := range draft.scenes {
		if !scene.pinned {
			continue
		}

		_, err := draft.renderScene(templates, scene, ContextLevelFull)
		if err != nil {
			return err
		}

		scene.level = ContextLevelFull
	}

	return nil
}

// renderScene renders a scene at a level once, and returns the same render after.
func (draft *contextDraft) renderScene(
	templates *template.Template, scene *draftScene, level string,
) (*sceneRender, error) {
	if render, ok := scene.renders[level]; ok {
		return render, nil
	}

	text, err := execute(templates, "scene", scene.view(level))
	if err != nil {
		return nil, err
	}

	scene.renders[level] = &sceneRender{text: text, size: 1 + utf8.RuneCountInString(text)}

	return scene.renders[level], nil
}

// neighborhood renders the scenes that enter the context, in narrative order.
func (draft *contextDraft) neighborhood(templates *template.Template) (string, error) {
	var scenes []string

	for _, scene := range draft.scenes {
		if scene.level != ContextLevelOmitted {
			scenes = append(scenes, scene.renders[scene.level].text)
		}
	}

	return execute(templates, "neighborhood", scenes)
}

// showTargetContext shows, on each side of the range of every manuscript target, the blocks nearest it while
// they fit in budget, in characters. It takes one block per side of every target in turn, so the nearest
// blocks of all targets come first. A side stops at the first block that does not fit, so its context stays
// contiguous.
func (draft *contextDraft) showTargetContext(templates *template.Template, budget int) error {
	var sides []*contextSide

	for _, target := range draft.targets {
		if target.span != nil {
			sides = append(sides, &contextSide{target: target, before: true}, &contextSide{target: target})
		}
	}

	for len(sides) > 0 {
		var open []*contextSide

		for _, side := range sides {
			block := side.next()
			if block == nil {
				continue
			}

			text, err := execute(templates, "block", block)
			if err != nil {
				return err
			}

			size := 1 + utf8.RuneCountInString(text)
			if size > budget {
				continue
			}

			budget -= size

			open = append(open, side)

			if side.before {
				side.target.before++
			} else {
				side.target.after++
			}
		}

		sides = open
	}

	return nil
}

// next returns the block the side shows next, outward from the range, or nil past the edge of the cell.
func (side *contextSide) next() *contentBlock {
	blocks, span := side.target.cell.content.Blocks, side.target.span

	position := span.End + side.target.after
	if side.before {
		position = span.Start - side.target.before - 1
	}

	if position < 0 || position >= len(blocks) {
		return nil
	}

	return blocks[position]
}

// allocate gives each ranked scene the richest level that fits in budget, in characters. Most relevant
// first, every scene takes its outline while the budget allows; then, in the same order, each takes the
// richest level that still fits. A scene too large for what remains is passed over, and a smaller one after
// it may still fit, so a block is never cut.
func (draft *contextDraft) allocate(templates *template.Template, budget int) error {
	for _, scene := range draft.ranked {
		outline, err := draft.renderScene(templates, scene, ContextLevelOutline)
		if err != nil {
			return err
		}

		if outline.size <= budget {
			scene.level = ContextLevelOutline
			budget -= outline.size
		}
	}

	for _, scene := range draft.ranked {
		if scene.level != ContextLevelOutline {
			continue
		}

		var err error

		budget, err = draft.upgrade(templates, scene, budget)
		if err != nil {
			return err
		}
	}

	return nil
}

// upgrade raises an outlined scene to the richest level that fits in budget, and returns the budget left.
// A level renders only when the text it adds at least may fit, so a large project renders few scenes whole.
func (draft *contextDraft) upgrade(templates *template.Template, scene *draftScene, budget int) (int, error) {
	outline := scene.renders[ContextLevelOutline].size

	for _, level := range []string{ContextLevelFull, ContextLevelRecap} {
		least, ok := scene.adds(level)
		if !ok || least > budget {
			continue
		}

		render, err := draft.renderScene(templates, scene, level)
		if err != nil {
			return 0, err
		}

		if render.size-outline <= budget {
			scene.level = level

			return budget - (render.size - outline), nil
		}
	}

	return budget, nil
}

// adds returns the characters a level adds at least to the scene's outline: the text of its manuscript
// cells, whole or as their fresh recaps. It reports false when the scene cannot enter at the level, as a
// recap without a fresh one.
func (scene *draftScene) adds(level string) (int, bool) {
	var (
		size   int
		recaps bool
	)

	for _, cell := range scene.cells {
		if cell.layer.layer.Kind != LayerKindManuscript {
			continue
		}

		switch {
		case level == ContextLevelFull:
			size += cell.content.size()
		case cell.recap != nil:
			size += utf8.RuneCountInString(cell.recap.Summary)
			recaps = true
		}
	}

	return size, level == ContextLevelFull || recaps
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
// revise mode. A manuscript target shows the blocks of its range its mode shows, and the context blocks
// chosen around it.
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
		view.Range = target.inside()

		blocks, span := target.cell.content.Blocks, target.span
		view.HasBefore, view.Before = span.Start > 0, blocks[span.Start-target.before:span.Start]
		view.HasAfter, view.After = span.End < len(blocks), blocks[span.End:span.End+target.after]
		view.BeforeOmitted, view.AfterOmitted = span.Start-target.before, len(blocks)-span.End-target.after
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

// estimateTokens estimates how many tokens a model reads in a text of chars characters.
func estimateTokens(chars int) int {
	return (chars + charsPerToken - 1) / charsPerToken
}
