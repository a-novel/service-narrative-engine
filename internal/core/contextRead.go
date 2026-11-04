package core

import (
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

const (
	// cellReadSize caps the cells one query of a [cellStore] reads.
	cellReadSize = 500
	// readAheadMin is the least text, in characters, one read ahead covers. Past the edge of the budget,
	// scenes small enough to fit may each be read and then not fit, and they then share a few queries.
	readAheadMin = 1 << 14
)

// outlineLevels lists the one level the first pass of the allocator gives a scene.
var outlineLevels = []string{ContextLevelOutline}

// recapKey names one version of the recap of a cell, by its level.
type recapKey struct {
	cell    cellKey
	version string
}

// cellStore reads the content and the recap versions of a project's cells as the calls of a generation
// first need them, and keeps them for every call. Read runs one query, within the plan that built the store.
type cellStore struct {
	read func(keys []*dao.CellsSelectKey) ([]*dao.CellsSelectRow, error)

	contents map[cellKey]*cellContent
	recaps   map[recapKey]string
}

// fetch reads what the store lacks: the content of the cells contents names, and the recap versions recaps
// names. A cell gone since the snapshot reads as empty.
func (store *cellStore) fetch(contents []cellKey, recaps []recapKey) error {
	var keys []*dao.CellsSelectKey

	for _, key := range contents {
		if _, ok := store.contents[key]; !ok {
			keys = append(keys, &dao.CellsSelectKey{SceneID: key.sceneID, LayerID: key.layerID, Content: true})
		}
	}

	for _, key := range recaps {
		if _, ok := store.recaps[key]; !ok {
			keys = append(keys, &dao.CellsSelectKey{
				SceneID: key.cell.sceneID, LayerID: key.cell.layerID, Recap: key.version,
			})
		}
	}

	for chunk := range slices.Chunk(keys, cellReadSize) {
		err := store.readChunk(chunk)
		if err != nil {
			return err
		}
	}

	for _, asked := range keys {
		key := cellKey{asked.SceneID, asked.LayerID}

		if asked.Content && store.contents[key] == nil {
			store.contents[key] = &cellContent{}
		}

		if _, ok := store.recaps[recapKey{key, asked.Recap}]; asked.Recap != "" && !ok {
			store.recaps[recapKey{key, asked.Recap}] = ""
		}
	}

	return nil
}

// readChunk reads one query of cells into the store.
func (store *cellStore) readChunk(keys []*dao.CellsSelectKey) error {
	rows, err := store.read(keys)
	if err != nil {
		return fmt.Errorf("read cells: %w", err)
	}

	for _, row := range rows {
		key := cellKey{row.SceneID, row.LayerID}

		if row.Content != nil {
			store.contents[key], err = readContent(row.Content)
			if err != nil {
				return err
			}
		}

		if row.RecapVersion != "" {
			store.recaps[recapKey{key, row.RecapVersion}] = lo.FromPtr(row.Recap)
		}
	}

	return nil
}

// read gives the cells of contents their content, and reads the recap versions recaps names, in one query
// for what the store lacks.
func (draft *contextDraft) read(contents []*draftCell, recaps []recapKey) error {
	err := draft.store.fetch(lo.Map(contents, func(cell *draftCell, _ int) cellKey { return cell.key }), recaps)
	if err != nil {
		return err
	}

	for _, cell := range contents {
		cell.content = draft.store.contents[cell.key]
	}

	return nil
}

// readAhead reads what the first scene lacks at the first of levels that fits budget, and stops there when
// it lacks nothing. The same query reads what each following scene lacks at its own first level that fits,
// until the scenes read cover the budget, or [readAheadMin]. The allocator then renders its next scenes
// without a query each, and a scene this guess misses reads alone when it renders.
func (draft *contextDraft) readAhead(scenes []*draftScene, levels []string, budget int) error {
	var (
		contents []*draftCell
		recaps   []recapKey
		covered  int
	)

	for index, scene := range scenes {
		sceneContents, sceneRecaps, least, ok := draft.ahead(scene, levels, budget)
		if !ok {
			continue
		}

		if index == 0 && len(sceneContents)+len(sceneRecaps) == 0 {
			return nil
		}

		contents, recaps = append(contents, sceneContents...), append(recaps, sceneRecaps...)

		covered += least
		if covered > max(budget, readAheadMin) {
			break
		}
	}

	return draft.read(contents, recaps)
}

// ahead returns what a scene lacks at the first of levels that fits budget, with that level's least size.
// A recap level brings the shorter versions that fit after it, which a render past its least size falls
// back to. It reports false when no level fits.
func (draft *contextDraft) ahead(
	scene *draftScene, levels []string, budget int,
) ([]*draftCell, []recapKey, int, bool) {
	for index, level := range levels {
		least, ok := scene.least(level)
		if !ok || least > budget {
			continue
		}

		contents, recaps := draft.unread(scene, level)

		if slices.Contains(recapLevels, level) {
			for _, next := range levels[index+1:] {
				if size, ok := scene.least(next); ok && size <= budget {
					_, nextRecaps := draft.unread(scene, next)
					recaps = append(recaps, nextRecaps...)
				}
			}
		}

		return contents, recaps, least, true
	}

	return nil, nil, 0, false
}

// unreadAt lists what scenes lack to render at a level, for one read.
func (draft *contextDraft) unreadAt(scenes []*draftScene, level string) ([]*draftCell, []recapKey) {
	var (
		contents []*draftCell
		recaps   []recapKey
	)

	for _, scene := range scenes {
		sceneContents, sceneRecaps := draft.unread(scene, level)
		contents, recaps = append(contents, sceneContents...), append(recaps, sceneRecaps...)
	}

	return contents, recaps
}

// unread lists the cells a scene shows whole at a level whose content the draft has not read, and the
// recap versions it shows that the store has not read. A target's cell shows as its handle, and needs
// neither.
func (draft *contextDraft) unread(scene *draftScene, level string) ([]*draftCell, []recapKey) {
	var (
		contents []*draftCell
		recaps   []recapKey
	)

	for _, shown := range scene.shows(level) {
		switch {
		case shown.cell.target != nil:
		case shown.version != "":
			key := recapKey{shown.cell.key, shown.version}
			if _, read := draft.store.recaps[key]; !read {
				recaps = append(recaps, key)
			}
		case shown.cell.content == nil:
			contents = append(contents, shown.cell)
		}
	}

	return contents, recaps
}

// least returns the characters the scene's render takes at least at a level, which a level must fit before
// it renders. An outline holds its markup and the text of the cells it shows, and a richer level adds to the
// outline what [draftScene.adds] counts. It reports false for a level the scene cannot take.
func (scene *draftScene) least(level string) (int, bool) {
	if level != ContextLevelOutline {
		return scene.adds(level)
	}

	size := scene.shell

	for _, shown := range scene.shows(level) {
		size += shown.cell.size()
	}

	return size, true
}

// size returns how many characters the cell's text holds, before any markup: its summary's count until the
// draft reads it.
func (cell *draftCell) size() int {
	if cell.content == nil {
		return cell.summary.Size
	}

	return cell.content.size()
}

// speakers returns the speakers of the blocks the cell shows the model: those its summary lists until the
// draft reads it, and for a manuscript target those around and inside its range the target shows.
func (cell *draftCell) speakers() []string {
	if cell.content == nil {
		return cell.summary.Speakers
	}

	blocks := cell.content.Blocks

	if target := cell.target; target != nil && target.span != nil {
		blocks = slices.Concat(blocks[:target.span.Start], target.inside(), blocks[target.span.End:])
	}

	return lo.Map(blocks, func(block *contentBlock, _ int) string { return block.Speaker })
}

// empty reports whether a cell holds no content, as its summary says until the draft reads it.
func (cell *draftCell) empty() bool {
	if cell.content == nil {
		return cell.summary.Empty
	}

	return cell.content.Text == "" && len(cell.content.Blocks) == 0
}
