package core

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

// charsPerToken is how many characters [estimateTokens] counts as one token. The planner measures every
// part of a prompt in characters, which add up exactly, and estimates tokens once.
const charsPerToken = 4

// frameShare divides a prompt's budget into the most its frame takes. Below that share the frame shows every
// beat whole, so it stays the same, and caches, across a project's calls.
const frameShare = 2

// sceneRender is a scene rendered at one level. Size counts its characters with the line it takes in the
// neighborhood.
type sceneRender struct {
	text string
	size int
}

// shownCell is a cell a scene shows at a level: whole, or as the version of its recap that names a level.
type shownCell struct {
	cell    *draftCell
	version string
}

// contextSide is one side of a manuscript target's range, where the task shows context blocks.
type contextSide struct {
	target *draftTarget
	before bool
}

// The prompt views are the data the templates of package prompts render. Handles stand for identifiers.
type (
	// promptFrame is the frame, with its beats rendered one per line.
	promptFrame struct {
		Title    string
		Premise  string
		Language string
		Layers   []*promptLayer
		Beats    []string
	}

	// promptFrameBeat is a beat of the frame at its level: whole, with the lines of its scenes, or as a
	// version of its recap, or as its title. A beat short of whole names the handles of its scenes.
	promptFrameBeat struct {
		Title       string
		Level       string
		Description string
		Lines       []*promptFrameScene
		Handles     string
		Recap       string
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

	// promptScene is a scene of the neighborhood. The scene of a target shows the prompts of its beat and its
	// own.
	promptScene struct {
		Handle     string
		Level      string
		BeatPrompt string
		Prompt     string
		Cells      []*promptCell
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
		Beat        *promptRecapBeat
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

	// promptRecapBeat is a beat to recap: its description, and each scene's manuscript as its fresh compact
	// recap, or whole.
	promptRecapBeat struct {
		Title       string
		Description string
		Scenes      []*promptRecapScene
	}

	promptRecapScene struct {
		Handle  string
		Recap   string
		Content *cellContent
	}

	promptSpan struct {
		First string
		Last  string
	}
)

// render plans what the call reads within the budget of its kind, and renders its prompt with the
// generation's templates and strict output schema.
//
// The frame, the task and the scenes that always enter come first. They may outgrow the budget up to the
// kind's limit, or its long limit as long context. A suggestion's rewrite fits half the output cap, or the
// whole cap for a single text or block. [ErrContextTooLarge] refuses a call past either. The blocks around
// each target's range come next, nearest first, and the other scenes share what is left.
func (draft *contextDraft) render(
	templates *template.Template, schema json.RawMessage, kind config.GenerationKind,
) (*ContextCall, error) {
	err := draft.checkRewrite(kind)
	if err != nil {
		return nil, err
	}

	prompt := &Prompt{
		Model: kind.Model, Reasoning: kind.Reasoning, MaxOutputTokens: kind.MaxOutputTokens, Schema: schema,
	}

	err = draft.renderSections(templates, prompt, kind)
	if err != nil {
		return nil, err
	}

	call := draft.call()
	call.Prompt = prompt
	call.Estimate = estimateTokens(prompt.size())

	return call, nil
}

// checkRewrite refuses a suggest call whose rewrite outgrows its share of the output cap: half the cap,
// the rest left for reasoning, or the whole cap for a single text or block.
func (draft *contextDraft) checkRewrite(kind config.GenerationKind) error {
	if draft.request.Kind != GenerationKindSuggest {
		return nil
	}

	share := kind.MaxOutputTokens / outputShare
	if len(draft.targets) == 1 && draft.targets[0].single() {
		share = kind.MaxOutputTokens
	}

	rewrite := estimateTokens(lo.SumBy(draft.targets, (*draftTarget).rewrite))
	if rewrite > share {
		return fmt.Errorf("%w: a rewrite of %d tokens, over the %d the output takes", ErrContextTooLarge, rewrite, share)
	}

	return nil
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

	// The frame takes at most its share of the budget, and never what the parts that always enter need.
	budget := kind.Budget * charsPerToken

	prompt.Frame, err = draft.renderFrame(templates, min(budget/frameShare, budget-prompt.size()))
	if err != nil {
		return err
	}

	// Past the kind's limit, the call enters as long context when the kind allows it.
	limit := max(kind.Limit, kind.LongLimit)
	if prompt.size() > limit*charsPerToken {
		return fmt.Errorf("%w: %d tokens, over the limit of %d", ErrContextTooLarge, estimateTokens(prompt.size()), limit)
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

	// A check reads its scenes, so only a generation that writes or recaps shows the author's prompts.
	text, err := execute(templates, "scene", scene.view(level, scene.pinned && draft.request.Kind != GenerationKindCheck))
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

	for _, level := range append([]string{ContextLevelFull}, recapLevels...) {
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
// cells, whole or as a version of their fresh recaps. It reports false when the scene cannot enter at the
// level, as a recap level without a fresh recap.
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
			size += utf8.RuneCountInString(cell.recap.version(level))
			recaps = true
		}
	}

	return size, level == ContextLevelFull || recaps
}

// shows lists the cells the scene shows at a level. A full scene shows every cell, a target's own cell as
// its handle. A condensed, compact or essentials scene shows its manuscript as that version of its fresh
// recaps, and an outline scene leaves it out. Every level shows the other cells whole.
func (scene *draftScene) shows(level string) []*shownCell {
	var shown []*shownCell

	for _, cell := range scene.cells {
		manuscript := cell.layer.layer.Kind == LayerKindManuscript

		switch {
		case level == ContextLevelFull, !manuscript:
			shown = append(shown, &shownCell{cell: cell})
		case cell.recap != nil && slices.Contains(recapLevels, level):
			shown = append(shown, &shownCell{cell: cell, version: level})
		}
	}

	return shown
}

// view returns what the template renders for the scene at a level, with the author's prompts for its beat
// and for itself when prompts is set.
func (scene *draftScene) view(level string, prompts bool) *promptScene {
	view := &promptScene{Handle: scene.handle, Level: level}
	if prompts {
		view.BeatPrompt, view.Prompt = scene.beat.beat.Prompt, scene.scene.Prompt
	}

	for _, shown := range scene.shows(level) {
		cell := &promptCell{Layer: shown.cell.layer.handle}

		switch {
		case shown.cell.target != nil:
			cell.Target = shown.cell.target.handle
		case shown.version != "":
			cell.Recap = shown.cell.recap.version(shown.version)
		default:
			cell.Content = shown.cell.content
		}

		view.Cells = append(view.Cells, cell)
	}

	return view
}

// renderFrame renders the frame within budget, in characters: the project, the layers the context shows,
// and every beat at the richest level that fits. A beat holding a target shows whole. Every other beat
// first takes its outline, which names its scenes' handles, then, most relevant first, the richest level
// that still fits. A frame within budget shows every beat whole, so it caches across a project's calls.
func (draft *contextDraft) renderFrame(templates *template.Template, budget int) (string, error) {
	for _, beat := range draft.beats {
		beat.level = lo.Ternary(beat.pinned, ContextLevelFull, ContextLevelOutline)

		render, err := draft.renderBeat(templates, beat, beat.level)
		if err != nil {
			return "", err
		}

		budget -= render.size
	}

	bare, err := execute(templates, "frame", draft.frameView(nil))
	if err != nil {
		return "", err
	}

	budget -= utf8.RuneCountInString(bare)

	for _, beat := range draft.rankedBeats {
		budget, err = draft.upgradeBeat(templates, beat, budget)
		if err != nil {
			return "", err
		}
	}

	beats := lo.Map(draft.beats, func(beat *draftBeat, _ int) string { return beat.renders[beat.level].text })

	return execute(templates, "frame", draft.frameView(beats))
}

// upgradeBeat raises an outlined beat to the richest level that fits in budget: whole, or a version of its
// fresh recap. It returns the budget left.
func (draft *contextDraft) upgradeBeat(templates *template.Template, beat *draftBeat, budget int) (int, error) {
	outline := beat.renders[ContextLevelOutline].size

	for _, level := range []string{ContextLevelFull, ContextLevelCompact, ContextLevelEssentials} {
		if level != ContextLevelFull && beat.recap == nil {
			continue
		}

		render, err := draft.renderBeat(templates, beat, level)
		if err != nil {
			return 0, err
		}

		if render.size-outline <= budget {
			beat.level = level

			return budget - (render.size - outline), nil
		}
	}

	return budget, nil
}

// renderBeat renders a beat's line of the frame at a level once, and returns the same render after.
func (draft *contextDraft) renderBeat(
	templates *template.Template, beat *draftBeat, level string,
) (*sceneRender, error) {
	if render, ok := beat.renders[level]; ok {
		return render, nil
	}

	text, err := execute(templates, "beat", draft.beatView(beat, level))
	if err != nil {
		return nil, err
	}

	beat.renders[level] = &sceneRender{text: text, size: 1 + utf8.RuneCountInString(text)}

	return beat.renders[level], nil
}

// frameView returns what the template renders for the frame around its rendered beats.
func (draft *contextDraft) frameView(beats []string) *promptFrame {
	view := &promptFrame{
		Title: draft.project.Title, Premise: draft.project.Premise, Language: draft.project.Language, Beats: beats,
	}

	for _, layer := range draft.layers {
		if layer.excluded && !layer.target {
			continue
		}

		view.Layers = append(view.Layers, &promptLayer{
			Handle: layer.handle, Kind: layer.layer.Kind, Name: layer.layer.Name, Description: layer.layer.Description,
		})
	}

	return view
}

// beatView returns what the template renders for a beat at a level. A whole beat lists its scenes, and
// hides a scene field the generation writes anew.
func (draft *contextDraft) beatView(beat *draftBeat, level string) *promptFrameBeat {
	view := &promptFrameBeat{Title: beat.beat.Title, Level: level}

	if level != ContextLevelFull {
		view.Handles = handleSpan(beat.scenes)
		if beat.recap != nil && level != ContextLevelOutline {
			view.Recap = beat.recap.version(level)
		}

		return view
	}

	view.Description = beat.beat.Description

	for _, scene := range beat.scenes {
		line := &promptFrameScene{Handle: scene.handle, Title: scene.scene.Title, Description: scene.scene.Description}

		for _, target := range draft.targets {
			if target.scene != scene || target.mode != TargetModeWrite {
				continue
			}

			switch target.field {
			case SceneFieldTitle:
				line.Title = ""
			case SceneFieldDescription:
				line.Description = ""
			}
		}

		view.Lines = append(view.Lines, line)
	}

	return view
}

// handleSpan names the handles of a run of scenes, as S12-S21, or S12 for one scene.
func handleSpan(scenes []*draftScene) string {
	switch len(scenes) {
	case 0:
		return ""
	case 1:
		return scenes[0].handle
	default:
		return scenes[0].handle + "-" + scenes[len(scenes)-1].handle
	}
}

// taskView returns what the template renders for the task of the generation's kind.
func (draft *contextDraft) taskView() *promptTask {
	view := &promptTask{Instruction: draft.request.Instruction}

	switch draft.request.Kind {
	case GenerationKindSuggest:
		view.Targets = lo.Map(draft.targets, func(target *draftTarget, _ int) *promptTarget { return target.view() })
	case GenerationKindConsolidate:
		target := draft.targets[0]
		if target.beat != nil {
			view.Beat = target.beatView()
		} else {
			view.Cell = &promptRecapCell{
				Scene: target.scene.handle, Layer: target.cell.layer.handle, Content: target.cell.content,
			}
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

// beatView returns what the template renders for the beat a consolidation recaps. Each scene shows its
// manuscript as its fresh compact recap where one exists, or whole.
func (target *draftTarget) beatView() *promptRecapBeat {
	view := &promptRecapBeat{Title: target.beat.beat.Title, Description: target.beat.beat.Description}

	for _, scene := range target.beat.scenes {
		line := &promptRecapScene{Handle: scene.handle}

		for _, cell := range scene.cells {
			if cell.target != target {
				continue
			}

			if cell.recap != nil {
				line.Recap = cell.recap.Compact
			} else {
				line.Content = cell.content
			}
		}

		view.Scenes = append(view.Scenes, line)
	}

	return view
}

// call returns the call the draft renders.
func (draft *contextDraft) call() *ContextCall {
	call := &ContextCall{}

	call.Beats = lo.Map(draft.beats, func(beat *draftBeat, _ int) *ContextBeat {
		return &ContextBeat{BeatID: beat.beat.ID, Level: beat.level}
	})

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
					LayerID: shown.cell.layer.layer.ID, Recap: shown.version,
					Override: shown.cell.override, UpdatedAt: shown.cell.updatedAt,
				})
			}
		}

		call.Scenes = append(call.Scenes, planned)
	}

	for _, layer := range draft.layers {
		if !layer.excluded || layer.target {
			call.Layers = append(call.Layers, &ContextLayer{Handle: layer.handle, LayerID: layer.layer.ID})
		}
	}

	call.Targets = lo.Map(draft.targets, func(target *draftTarget, _ int) *ContextTarget { return target.plan() })

	return call
}

// plan returns the plan of the target.
func (target *draftTarget) plan() *ContextTarget {
	if target.beat != nil {
		return &ContextTarget{
			Handle: target.handle, BeatID: target.beat.beat.ID, UpdatedAt: target.beat.beat.UpdatedAt,
			Sources: target.sources,
		}
	}

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
