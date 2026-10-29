package handlers

import (
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// BeatPage is the REST representation of a page of a project's beats.
type BeatPage struct {
	Beats []Beat `json:"beats"`

	// Total counts every beat of the project, so a client paging through them knows when it holds
	// them all.
	Total int `json:"total"`
}

func loadBeatPage(page *core.BeatPage) BeatPage {
	return BeatPage{
		Beats: lo.Map(page.Beats, func(beat *core.Beat, _ int) Beat { return loadBeat(beat) }),
		Total: page.Total,
	}
}
