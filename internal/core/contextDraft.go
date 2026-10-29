package core

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"

	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

// nameWeight is how much each name a scene shares with the focus adds to its relevance. A scene sharing
// two names with the focus weighs as much as the scene next to it.
const nameWeight = 0.25

// elisions lists the French words that drop their vowel before another word, as in l'Ombre or d'Artagnan.
var elisions = []string{"c", "d", "j", "l", "m", "n", "s", "t", "qu", "jusqu", "lorsqu", "puisqu"}

// cellContent is the content of a cell as the planner reads it: the text of a notes cell, or the blocks
// of a manuscript cell.
type cellContent struct {
	Text   string          `json:"text"`
	Blocks []*contentBlock `json:"blocks"`
}

// contentBlock is one block of a manuscript cell. Handle and Kept say how a target shows it.
type contentBlock struct {
	Kind    string         `json:"kind"`
	Text    string         `json:"text"`
	Speaker string         `json:"speaker"`
	Parts   []*contentPart `json:"parts"`

	Handle string `json:"-"`
	Kept   bool   `json:"-"`
}

// contentPart is one part of a dialogue block: words spoken, or the tone of the next ones.
type contentPart struct {
	Text string `json:"text"`
	Tone string `json:"tone"`
}

// cellKey identifies a cell by its scene and its layer.
type cellKey struct {
	sceneID uuid.UUID
	layerID uuid.UUID
}

// targetKey identifies what a target writes: a cell, or a field of a scene.
type targetKey struct {
	cellKey

	field string
}

// draftLayer is a layer as a generation reads it.
type draftLayer struct {
	layer  *Layer
	handle string

	// excluded marks a layer the request does not read.
	excluded bool
	// target marks the layer of a target, which the frame lists even when excluded.
	target bool
}

// draftCell is a cell as a generation reads it, with the request's override applied.
type draftCell struct {
	layer   *draftLayer
	content *cellContent

	// recap is the cell's fresh recap, nil when it has none.
	recap *CellRecap
	// target is the target that writes or recaps the cell, nil when none does.
	target *draftTarget

	override  bool
	updatedAt time.Time
}

// draftScene is a scene as a generation reads it, with the request's override applied.
type draftScene struct {
	scene  *Scene
	handle string

	// cells lists the cells the scene shows, by layer order: those of the layers the request reads, and
	// those of its targets.
	cells []*draftCell
	// pinned marks the scene of a target, which always enters whole.
	pinned bool
	// focus marks a scene the generation is about: the scene of a target, or a scene to check.
	focus bool
	// names holds the folded names the scene's shown content uses.
	names map[string]bool

	// level is the level the scene enters at, and renders holds the scene rendered at each level it can
	// enter at.
	level   string
	renders map[string]*sceneRender
	score   float64

	override bool
}

// draftTarget is a target as a generation reads it.
type draftTarget struct {
	handle string
	scene  *draftScene
	// cell is the target's cell, nil for a scene field.
	cell  *draftCell
	field string
	mode  string

	// span is the range of a manuscript target, and kept the positions of its kept blocks.
	span *Range
	kept []int
	// blocks maps the handle of each block shown inside the range to the block's position.
	blocks []*ContextBlock
}

// contextDraft is a project as one generation reads it, while the planner builds the generation's
// context: the snapshot with the request's overrides and exclusions applied, and a handle for every scene,
// layer, target and block the model may see.
type contextDraft struct {
	request *GenerationPlanRequest
	project *Project

	// scenes lists every scene in narrative order, and layers every layer in order.
	scenes []*draftScene
	layers []*draftLayer
	// targets lists the targets in the order of the request. A consolidate generation's cell is its only
	// target.
	targets []*draftTarget
	// ranked lists the scenes without a target, most relevant first.
	ranked []*draftScene

	sceneByID map[uuid.UUID]*draftScene
	layerByID map[uuid.UUID]*draftLayer
	cells     map[cellKey]*draftCell
}

// newContextDraft reads the snapshot of a request as the generation sees it, and ranks the scenes the
// generation does not write in.
func newContextDraft(request *GenerationPlanRequest) (*contextDraft, error) {
	draft := &contextDraft{request: request, project: request.Snapshot.Project}
	draft.layers, draft.layerByID = readLayers(request.Snapshot.Layers)
	draft.scenes, draft.sceneByID = readScenes(request.Snapshot.Scenes)

	err := draft.applyOverrides()
	if err != nil {
		return nil, err
	}

	draft.cells, err = draft.readCells()
	if err != nil {
		return nil, err
	}

	draft.targets, err = draft.readTargets()
	if err != nil {
		return nil, err
	}

	err = draft.readExclusions()
	if err != nil {
		return nil, err
	}

	draft.collectCells()
	draft.ranked = draft.rank()

	return draft, nil
}

// readLayers reads the layers in order, each with its handle, and indexes them by ID.
func readLayers(layers []*Layer) ([]*draftLayer, map[uuid.UUID]*draftLayer) {
	ordered := slices.Clone(layers)
	slices.SortStableFunc(ordered, func(a, b *Layer) int { return cmp.Compare(a.Position, b.Position) })

	read := make([]*draftLayer, len(ordered))
	byID := make(map[uuid.UUID]*draftLayer, len(ordered))

	for index, layer := range ordered {
		read[index] = &draftLayer{layer: layer, handle: handle("L", index)}
		byID[layer.ID] = read[index]
	}

	return read, byID
}

// readScenes reads the scenes in narrative order, each with its handle, and indexes them by ID.
func readScenes(scenes []*Scene) ([]*draftScene, map[uuid.UUID]*draftScene) {
	ordered := slices.Clone(scenes)
	slices.SortStableFunc(ordered, func(a, b *Scene) int { return cmp.Compare(a.Position, b.Position) })

	read := make([]*draftScene, len(ordered))
	byID := make(map[uuid.UUID]*draftScene, len(ordered))

	for index, scene := range ordered {
		read[index] = &draftScene{scene: scene, handle: handle("S", index), level: ContextLevelOmitted}
		byID[scene.ID] = read[index]
	}

	return read, byID
}

// applyOverrides replaces the fields of the scenes the request overrides.
func (draft *contextDraft) applyOverrides() error {
	if draft.request.Overrides == nil {
		return nil
	}

	for _, override := range draft.request.Overrides.Scenes {
		read, ok := draft.sceneByID[override.ID]
		if !ok {
			return fmt.Errorf("%w: override of %s", ErrSceneNotFound, override.ID)
		}

		scene := *read.scene
		scene.Title, scene.Description, scene.Prompt = override.Title, override.Description, override.Prompt
		read.scene, read.override = &scene, true
	}

	return nil
}

// readCells reads the content of every cell, keyed by scene and layer, the request's overrides applied.
// Each cell keeps its fresh recap.
func (draft *contextDraft) readCells() (map[cellKey]*draftCell, error) {
	cells := make(map[cellKey]*draftCell, len(draft.request.Snapshot.Cells))

	for _, cell := range draft.request.Snapshot.Cells {
		read, err := draft.newCell(cell.SceneID, cell.LayerID, cell.Content)
		if err != nil {
			return nil, err
		}

		read.updatedAt = cell.UpdatedAt
		cells[cellKey{cell.SceneID, cell.LayerID}] = read
	}

	err := draft.readCellOverrides(cells)
	if err != nil {
		return nil, err
	}

	for _, recap := range draft.request.Snapshot.Recaps {
		cell, ok := cells[cellKey{recap.SceneID, recap.LayerID}]
		if ok && !cell.override && cell.updatedAt.Equal(recap.CellUpdatedAt) {
			cell.recap = recap
		}
	}

	return cells, nil
}

// readCellOverrides replaces in cells the content of the cells the request overrides, once it passes the
// content schema of the layer's kind.
func (draft *contextDraft) readCellOverrides(cells map[cellKey]*draftCell) error {
	if draft.request.Overrides == nil {
		return nil
	}

	overridden := make(map[cellKey]bool, len(draft.request.Overrides.Cells))

	for _, override := range draft.request.Overrides.Cells {
		key := cellKey{override.SceneID, override.LayerID}
		if overridden[key] {
			return fmt.Errorf("%w: two overrides of one cell", ErrInvalidRequest)
		}

		overridden[key] = true

		read, err := draft.newCell(override.SceneID, override.LayerID, override.Content)
		if err != nil {
			return err
		}

		schema, err := schemas.Content(read.layer.layer.Kind)
		if err != nil {
			return fmt.Errorf("content schema: %w", err)
		}

		err = schema.Validate(override.Content)
		if err != nil {
			return errors.Join(fmt.Errorf("cell override: %w", err), ErrInvalidRequest)
		}

		read.override = true
		cells[key] = read
	}

	return nil
}

// newCell reads the content of the cell of a scene in a layer.
func (draft *contextDraft) newCell(sceneID, layerID uuid.UUID, content json.RawMessage) (*draftCell, error) {
	if _, ok := draft.sceneByID[sceneID]; !ok {
		return nil, fmt.Errorf("%w: cell of %s", ErrSceneNotFound, sceneID)
	}

	layer, ok := draft.layerByID[layerID]
	if !ok {
		return nil, fmt.Errorf("%w: cell of %s", ErrLayerNotFound, layerID)
	}

	read := &draftCell{layer: layer, content: &cellContent{}}

	err := json.Unmarshal(content, read.content)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("read cell content: %w", err), ErrInvalidRequest)
	}

	return read, nil
}

// cell returns the cell of a scene in a layer, empty when it was never written.
func (draft *contextDraft) cell(scene *draftScene, layer *draftLayer) *draftCell {
	key := cellKey{scene.scene.ID, layer.layer.ID}

	read, ok := draft.cells[key]
	if !ok {
		read = &draftCell{layer: layer, content: &cellContent{}}
		draft.cells[key] = read
	}

	return read
}

// readTargets reads the targets of the generation's kind. A check generation has none, and marks the
// scenes it checks instead.
func (draft *contextDraft) readTargets() ([]*draftTarget, error) {
	switch draft.request.Kind {
	case GenerationKindSuggest:
		return draft.suggestTargets()
	case GenerationKindConsolidate:
		return draft.recapTarget()
	default:
		return nil, draft.markChecked()
	}
}

// suggestTargets reads the targets of a suggest generation, each with the handles of the blocks it shows.
func (draft *contextDraft) suggestTargets() ([]*draftTarget, error) {
	if len(draft.request.Targets) == 0 {
		return nil, fmt.Errorf("%w: a suggest generation needs a target", ErrInvalidRequest)
	}

	targets := make([]*draftTarget, len(draft.request.Targets))
	written := make(map[targetKey]bool, len(draft.request.Targets))

	var blocks int

	for index, request := range draft.request.Targets {
		key := targetKey{cellKey{request.SceneID, request.LayerID}, request.Field}
		if written[key] {
			return nil, fmt.Errorf("%w: target %d repeats another", ErrInvalidRequest, index)
		}

		written[key] = true

		target, err := draft.newTarget(request, handle("T", index), &blocks)
		if err != nil {
			return nil, fmt.Errorf("target %d: %w", index, err)
		}

		targets[index] = target
	}

	return targets, nil
}

// newTarget reads the scene field or the cell a target writes, under the handle name, and pins its scene.
// Blocks counts the block handles the targets before it took.
func (draft *contextDraft) newTarget(request *GenerationTarget, name string, blocks *int) (*draftTarget, error) {
	scene, ok := draft.sceneByID[request.SceneID]
	if !ok {
		return nil, ErrSceneNotFound
	}

	target := &draftTarget{handle: name, scene: scene, field: request.Field, mode: request.Mode}
	scene.pinned, scene.focus = true, true

	if request.Field != "" {
		return target, nil
	}

	layer, ok := draft.layerByID[request.LayerID]
	if !ok {
		return nil, ErrLayerNotFound
	}

	target.cell = draft.cell(scene, layer)
	target.cell.target, layer.target = target, true

	var err error

	target.span, target.blocks, err = targetRange(target.cell, request, blocks)
	if err != nil {
		return nil, err
	}

	target.kept = request.Kept

	return target, nil
}

// targetRange checks the range and the kept blocks of a target on a cell, and gives a handle to each block
// the target shows inside its range: every block in revise mode, the kept ones in write mode. It returns
// the range, the whole cell when the request names none, and the handles. Only a manuscript target takes
// a range or kept blocks, and a text target has neither.
func targetRange(cell *draftCell, request *GenerationTarget, blocks *int) (*Range, []*ContextBlock, error) {
	if cell.layer.layer.Kind != LayerKindManuscript {
		if request.Range != nil || len(request.Kept) > 0 {
			return nil, nil, fmt.Errorf("%w: a range or kept blocks on a text target", ErrInvalidRequest)
		}

		return nil, nil, nil
	}

	content := cell.content.Blocks
	span := &Range{Start: 0, End: len(content)}

	if request.Range != nil {
		if request.Range.End > len(content) {
			return nil, nil, fmt.Errorf("%w: range ends past the cell's %d blocks", ErrInvalidRequest, len(content))
		}

		span = request.Range
	}

	for _, position := range request.Kept {
		if position < span.Start || position >= span.End {
			return nil, nil, fmt.Errorf("%w: kept block %d is outside the range", ErrInvalidRequest, position)
		}

		content[position].Kept = true
	}

	var handles []*ContextBlock

	for position := span.Start; position < span.End; position++ {
		if request.Mode == TargetModeWrite && !content[position].Kept {
			continue
		}

		content[position].Handle = handle("B", *blocks)
		handles = append(handles, &ContextBlock{Handle: content[position].Handle, Position: position})
		*blocks++
	}

	return span, handles, nil
}

// shown returns the blocks of a manuscript target the model sees: those before its range, those inside it,
// and those after it. In write mode the range shows its kept blocks alone.
func (target *draftTarget) shown() ([]*contentBlock, []*contentBlock, []*contentBlock) {
	content := target.cell.content.Blocks

	var inside []*contentBlock

	for _, block := range content[target.span.Start:target.span.End] {
		if target.mode == TargetModeRevise || block.Kept {
			inside = append(inside, block)
		}
	}

	return content[:target.span.Start], inside, content[target.span.End:]
}

// recapTarget reads the cell a consolidate generation recaps, as its only target.
func (draft *contextDraft) recapTarget() ([]*draftTarget, error) {
	scene, ok := draft.sceneByID[draft.request.Cell.SceneID]
	if !ok {
		return nil, ErrSceneNotFound
	}

	layer, ok := draft.layerByID[draft.request.Cell.LayerID]
	if !ok {
		return nil, ErrLayerNotFound
	}

	target := &draftTarget{handle: handle("T", 0), scene: scene, cell: draft.cell(scene, layer)}
	target.cell.target, layer.target = target, true
	scene.pinned, scene.focus = true, true

	return []*draftTarget{target}, nil
}

// markChecked marks the scenes a check generation checks: the range of the request, or every scene.
func (draft *contextDraft) markChecked() error {
	span := Range{Start: 0, End: len(draft.scenes)}

	if draft.request.Scenes != nil {
		span = *draft.request.Scenes
		if span.End > len(draft.scenes) || span.Start == span.End {
			return fmt.Errorf("%w: the scenes to check are outside the project's %d", ErrInvalidRequest, len(draft.scenes))
		}
	}

	for _, scene := range draft.scenes[span.Start:span.End] {
		scene.focus = true
	}

	return nil
}

// readExclusions marks the layers the request does not read.
func (draft *contextDraft) readExclusions() error {
	for _, layerID := range draft.request.Exclude {
		layer, ok := draft.layerByID[layerID]
		if !ok {
			return fmt.Errorf("%w: excluded %s", ErrLayerNotFound, layerID)
		}

		layer.excluded = true
	}

	return nil
}

// collectCells lists the cells each scene shows, and the names they use. A scene shows the cells of the
// layers the request reads that hold content, and the cells of its targets.
func (draft *contextDraft) collectCells() {
	for _, scene := range draft.scenes {
		scene.names = map[string]bool{}

		for _, layer := range draft.layers {
			cell, ok := draft.cells[cellKey{scene.scene.ID, layer.layer.ID}]
			if !ok || (cell.target == nil && (layer.excluded || cell.empty())) {
				continue
			}

			scene.cells = append(scene.cells, cell)
			cell.addNames(scene.names)
		}
	}
}

// empty reports whether a cell holds no content.
func (cell *draftCell) empty() bool {
	return cell.content.Text == "" && len(cell.content.Blocks) == 0
}

// addNames adds to names the folded names the cell shows the model: the speakers of its shown blocks and
// the names of its fresh recap.
func (cell *draftCell) addNames(names map[string]bool) {
	blocks := cell.content.Blocks

	if cell.target != nil && cell.target.span != nil {
		before, inside, after := cell.target.shown()
		blocks = slices.Concat(before, inside, after)
	}

	for _, block := range blocks {
		if block.Speaker != "" {
			names[foldName(block.Speaker)] = true
		}
	}

	if cell.recap != nil {
		for _, name := range cell.recap.Names {
			names[foldName(name)] = true
		}
	}
}

// rank scores each scene without a target by its relevance to the focus, and returns them most relevant
// first. Equal scores keep narrative order.
func (draft *contextDraft) rank() []*draftScene {
	focusNames := map[string]bool{}

	var focus []int

	for position, scene := range draft.scenes {
		if scene.focus {
			focus = append(focus, position)

			for name := range scene.names {
				focusNames[name] = true
			}
		}
	}

	var ranked []*draftScene

	for position, scene := range draft.scenes {
		if scene.pinned {
			continue
		}

		var shared int

		for name := range scene.names {
			if focusNames[name] {
				shared++
			}
		}

		scene.score = relevance(distance(position, focus), shared)
		ranked = append(ranked, scene)
	}

	slices.SortStableFunc(ranked, func(a, b *draftScene) int { return cmp.Compare(b.score, a.score) })

	return ranked
}

// relevance scores how much a scene matters to a generation, from its distance to the focus in narrative
// order and the names it shares with the focus. The allocator gives the richest levels to the highest
// scores.
func relevance(distance, sharedNames int) float64 {
	return 1/float64(1+distance) + nameWeight*float64(sharedNames)
}

// distance returns how many scenes separate position from the closest of the focus positions, or 0
// without a focus.
func distance(position int, focus []int) int {
	if len(focus) == 0 {
		return 0
	}

	closest := math.MaxInt

	for _, other := range focus {
		closest = min(closest, max(position-other, other-position))
	}

	return closest
}

// foldName returns the form of a name that matching compares: lowercase, without accents, and without the
// French elision that may start each word.
func foldName(name string) string {
	decomposed := norm.NFD.String(strings.ToLower(strings.ReplaceAll(name, "’", "'")))

	folded := strings.Map(func(char rune) rune {
		if unicode.Is(unicode.Mn, char) {
			return -1
		}

		return char
	}, decomposed)

	words := strings.Fields(folded)

	for index, word := range words {
		elided, rest, ok := strings.Cut(word, "'")
		if ok && slices.Contains(elisions, elided) {
			words[index] = rest
		}
	}

	return strings.Join(words, " ")
}

// handle names the item at index, from 0, among the items of one prefix: S1 is the first scene.
func handle(prefix string, index int) string {
	return prefix + strconv.Itoa(index+1)
}
