package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"text/template"

	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

// outputShare divides a call's output cap into the most a suggestion's rewrite takes, so the rest stays for
// reasoning. A call that rewrites a single text or block may take the whole cap.
const outputShare = 2

// windowShare divides a prompt's budget into the most the scenes of one check window take, so the frame and
// the context around the window share the rest.
const windowShare = 2

// contextSplitter plans the calls of one generation: a single call while it fits, or several when its kind
// splits it.
type contextSplitter struct {
	request *GenerationPlanRequest
	kind    config.GenerationKind

	// templates and schema render every call in the project's language.
	templates *template.Template
	schema    json.RawMessage
	// contents holds the content of each stored cell once read, for every call.
	contents map[cellKey]*cellContent
}

// calls plans the generation as one call, or splits it when its kind allows. A suggest generation that fits
// no single call gives each target a call of its own. A check whose scenes do not all enter whole checks
// them in windows.
func (splitter *contextSplitter) calls() ([]*ContextCall, error) {
	draft, call, err := splitter.call(splitter.request, &callPart{})

	split := splitter.kind.Split

	switch {
	case split && splitter.request.Kind == GenerationKindSuggest && errors.Is(err, ErrContextTooLarge):
		return splitter.splitTargets(draft)
	case err != nil:
		return nil, err
	case split && splitter.request.Kind == GenerationKindCheck && !draft.checkedWhole():
		return splitter.splitWindows(draft)
	default:
		return []*ContextCall{call}, nil
	}
}

// call plans one call of the generation from a request, for one part of it. It returns the draft it
// planned on, for a call too large too.
func (splitter *contextSplitter) call(
	request *GenerationPlanRequest, part *callPart,
) (*contextDraft, *ContextCall, error) {
	draft, err := newContextDraft(request, part, splitter.contents)
	if err != nil {
		return nil, nil, err
	}

	call, err := draft.render(splitter.templates, splitter.schema, splitter.kind)

	return draft, call, err
}

// splitTargets gives each target of a suggest generation a call of its own, in the order of the request. A
// target whose rewrite passes the output share takes a call for each span of its range. Every call names a
// target and its blocks as the whole generation would.
func (splitter *contextSplitter) splitTargets(draft *contextDraft) ([]*ContextCall, error) {
	var calls []*ContextCall

	share := splitter.kind.MaxOutputTokens / outputShare * charsPerToken
	part := &callPart{handles: map[string]int{}}

	for index, target := range draft.targets {
		for _, span := range target.spans(share) {
			_, call, err := splitter.call(splitter.targetRequest(index, span), part.at(target, span))
			if err != nil {
				return nil, fmt.Errorf("target %d: %w", index, err)
			}

			calls = append(calls, call)
		}

		part = part.after(target)
	}

	return calls, nil
}

// targetRequest returns the request of the call for one target, narrowed to a span of its range. A text
// target has no span.
func (splitter *contextSplitter) targetRequest(index int, span *Range) *GenerationPlanRequest {
	target := *splitter.request.Targets[index]
	if span != nil {
		target.Range = span
		target.Kept = lo.Filter(target.Kept, func(position, _ int) bool {
			return position >= span.Start && position < span.End
		})
	}

	request := *splitter.request
	request.Targets = []*GenerationTarget{&target}

	return &request
}

// splitWindows checks the scenes of a check in consecutive windows, each whole within its share of the
// budget.
func (splitter *contextSplitter) splitWindows(draft *contextDraft) ([]*ContextCall, error) {
	windows, err := draft.windows(splitter.templates, splitter.kind.Budget*charsPerToken/windowShare)
	if err != nil {
		return nil, err
	}

	calls := make([]*ContextCall, len(windows))

	for index, window := range windows {
		request := *splitter.request
		request.Scenes = window

		_, calls[index], err = splitter.call(&request, &callPart{window: true})
		if err != nil {
			return nil, fmt.Errorf("window %d: %w", index, err)
		}
	}

	return calls, nil
}

// at returns the part of the call for a span of a target, when the part's handles start at the target.
// Blocks keep their handles across spans, in revise mode, which shows every block of the range.
func (part *callPart) at(target *draftTarget, span *Range) *callPart {
	blocks := part.blocks
	if span != nil {
		blocks += span.Start - target.span.Start
	}

	return &callPart{handles: part.handles, blocks: blocks}
}

// after returns the part where the handles of the targets after a target start.
func (part *callPart) after(target *draftTarget) *callPart {
	prefix := targetPrefix(target)

	return &callPart{
		handles: lo.Assign(part.handles, map[string]int{prefix: part.handles[prefix] + 1}),
		blocks:  part.blocks + len(target.blocks),
	}
}

// checkedWhole reports whether every scene a check reads enters whole.
func (draft *contextDraft) checkedWhole() bool {
	return lo.EveryBy(draft.scenes, func(scene *draftScene) bool {
		return !scene.focus || scene.level == ContextLevelFull
	})
}

// windows divides the scenes a check reads into consecutive windows whose whole renders each fit in share
// characters. A scene past share takes a window of its own.
func (draft *contextDraft) windows(templates *template.Template, share int) ([]*Range, error) {
	var (
		windows []*Range
		size    int
	)

	for position, scene := range draft.scenes {
		if !scene.focus {
			continue
		}

		render, err := draft.renderScene(templates, scene, ContextLevelFull)
		if err != nil {
			return nil, err
		}

		if len(windows) == 0 || size+render.size > share {
			windows = append(windows, &Range{Start: position, End: position})
			size = 0
		}

		windows[len(windows)-1].End++
		size += render.size
	}

	return windows, nil
}
